package whois

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	rdapBootstrapURL = "https://data.iana.org/rdap/dns.json"
	ianaWhoisServer  = "whois.iana.org"
	whoisTimeout     = 4 * time.Second
	whoisCacheTTL    = time.Hour
	whoisCacheMax    = 1000
	maxWhoisResponse = 100_000
	maxLevels        = 4
	serverRetryAfter = 10 * time.Minute
)

var whoisFallbackServers = map[string][]string{
	"ru": {"whois.nic.ru"}, "su": {"whois.nic.ru"}, "xn--p1ai": {"whois.nic.ru"},
}

var whoisFields = map[string][]string{
	"registrar": {"registrar", "sponsoring registrar", "registrar name"},
	"created":   {"creation date", "created", "created on", "registered on", "registered", "registration time", "domain registration date"},
	"expires": {"registry expiry date", "registrar registration expiration date", "expiry date",
		"expiration date", "expiration time", "expires", "expires on", "paid-till", "renewal date"},
}

var dateLayouts = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02", "02-Jan-2006", "2006.01.02", "02.01.2006", "2006/01/02", "02/01/2006"}

var whoisHTTP = &http.Client{Timeout: whoisTimeout}

type Info struct {
	Domain    string
	Registrar string
	Created   time.Time
	Expires   time.Time
}

func parseDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	candidates := []string{value, strings.Fields(value)[0]}
	if len(value) >= 10 {
		candidates = append(candidates, value[:10])
	}
	for _, c := range candidates {
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, c); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

var (
	rdapMu      sync.Mutex
	rdapServers map[string]string
)

func getRDAPServers() map[string]string {
	rdapMu.Lock()
	defer rdapMu.Unlock()
	if rdapServers != nil {
		return rdapServers
	}
	var data struct {
		Services [][][]string `json:"services"`
	}
	if err := getJSON(rdapBootstrapURL, nil, &data); err != nil {
		log.Printf("RDAP bootstrap: %v", err)
		return nil
	}
	servers := map[string]string{}
	for _, s := range data.Services {
		if len(s) < 2 || len(s[1]) == 0 {
			continue
		}
		u := s[1][0]
		for _, c := range s[1] {
			if strings.HasPrefix(c, "https://") {
				u = c
				break
			}
		}
		for _, tld := range s[0] {
			servers[strings.ToLower(tld)] = strings.TrimRight(u, "/") + "/"
		}
	}
	rdapServers = servers
	return servers
}

type httpStatusError int

func (e httpStatusError) Error() string { return "HTTP " + http.StatusText(int(e)) }

func getJSON(url string, headers map[string]string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := whoisHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return httpStatusError(resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 5<<20)).Decode(out)
}

func parseRDAP(data map[string]any, domain string) *Info {
	events := map[string]string{}
	if list, ok := data["events"].([]any); ok {
		for _, e := range list {
			if m, ok := e.(map[string]any); ok {
				a, _ := m["eventAction"].(string)
				d, _ := m["eventDate"].(string)
				events[a] = d
			}
		}
	}
	info := &Info{Domain: domain, Created: parseDate(events["registration"]), Expires: parseDate(events["expiration"])}
	if n, _ := data["ldhName"].(string); n != "" {
		info.Domain = strings.ToLower(n)
	}
	entities, _ := data["entities"].([]any)
	for _, e := range entities {
		m, _ := e.(map[string]any)
		roles, _ := m["roles"].([]any)
		isRegistrar := false
		for _, r := range roles {
			if r == "registrar" {
				isRegistrar = true
			}
		}
		if !isRegistrar {
			continue
		}
		vcard, _ := m["vcardArray"].([]any)
		if len(vcard) > 1 {
			items, _ := vcard[1].([]any)
			for _, it := range items {
				item, _ := it.([]any)
				if len(item) > 3 && item[0] == "fn" {
					info.Registrar, _ = item[3].(string)
					break
				}
			}
		}
		break
	}
	return info
}

func rdapLookup(server, domain string) (*Info, error) {
	var data map[string]any
	err := getJSON(server+"domain/"+domain, map[string]string{"accept": "application/rdap+json"}, &data)
	if s, ok := err.(httpStatusError); ok && s == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseRDAP(data, domain), nil
}

func whoisQuery(server, query string) (string, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server, "43"), whoisTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(whoisTimeout))
	if _, err := conn.Write([]byte(query + "\r\n")); err != nil {
		return "", err
	}
	b, err := io.ReadAll(io.LimitReader(conn, maxWhoisResponse))
	if len(b) > 0 {
		err = nil
	}
	return string(b), err
}

var (
	whoisLineRe    = regexp.MustCompile(`(?im)^whois:\s*(\S+)`)
	whoisServersMu sync.Mutex
	whoisServerOf  = map[string]string{}
	serverDownTill = map[string]time.Time{}
)

func whoisServer(tld string) (string, error) {
	whoisServersMu.Lock()
	s, ok := whoisServerOf[tld]
	whoisServersMu.Unlock()
	if ok {
		return s, nil
	}
	text, err := whoisQuery(ianaWhoisServer, tld)
	if err != nil {
		return "", err
	}
	if m := whoisLineRe.FindStringSubmatch(text); m != nil {
		s = m[1]
	}
	whoisServersMu.Lock()
	whoisServerOf[tld] = s
	whoisServersMu.Unlock()
	return s, nil
}

func parseWhois(text, domain string) *Info {
	values := map[string]string{}
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		k, v, found := strings.Cut(line, ":")
		key, value := strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		indented := line != "" && (line[0] == ' ' || line[0] == '\t')
		switch {
		case !indented:
			section = ""
			if found && value == "" {
				section = key
			}
		case section != "":
			key = section + " " + key
		}
		if !found || value == "" {
			continue
		}
		for field, names := range whoisFields {
			if _, done := values[field]; done {
				continue
			}
			for _, n := range names {
				if n == key {
					values[field] = value
				}
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return &Info{Domain: domain, Registrar: values["registrar"], Created: parseDate(values["created"]), Expires: parseDate(values["expires"])}
}

func whoisLookup(domain, tld string) (*Info, error) {
	var lastErr error
	var servers []string
	if s, err := whoisServer(tld); err != nil {
		log.Printf("IANA whois for .%s: %v", tld, err)
		lastErr = err
	} else if s != "" {
		servers = append(servers, s)
	}
	servers = append(servers, whoisFallbackServers[tld]...)
	for _, s := range servers {
		whoisServersMu.Lock()
		down := time.Now().Before(serverDownTill[s])
		whoisServersMu.Unlock()
		if down {
			continue
		}
		text, err := whoisQuery(s, domain)
		if err != nil {
			log.Printf("WHOIS server %s: %v", s, err)
			whoisServersMu.Lock()
			serverDownTill[s] = time.Now().Add(serverRetryAfter)
			whoisServersMu.Unlock()
			lastErr = err
			continue
		}
		return parseWhois(text, domain), nil
	}
	return nil, lastErr
}

type whoisCacheEntry struct {
	at   time.Time
	info *Info
}

var (
	whoisCacheMu sync.Mutex
	whoisCache   = map[string]whoisCacheEntry{}
)

func Lookup(domain string) *Info {
	whoisCacheMu.Lock()
	c, ok := whoisCache[domain]
	whoisCacheMu.Unlock()
	if ok && time.Since(c.at) < whoisCacheTTL {
		return c.info
	}
	labels := strings.Split(strings.ToLower(domain), ".")
	tld := labels[len(labels)-1]
	var result *Info
	var failed bool
	server := getRDAPServers()[tld]
	for i := max(0, len(labels)-maxLevels); i < len(labels)-1; i++ {
		cand := strings.Join(labels[i:], ".")
		var err error
		if server != "" {
			result, err = rdapLookup(server, cand)
		} else {
			result, err = whoisLookup(cand, tld)
		}
		if err != nil {
			log.Printf("Domain lookup for %s: %v", cand, err)
			failed = true
			continue
		}
		if result != nil {
			break
		}
	}
	if result == nil && failed {
		return nil
	}
	whoisCacheMu.Lock()
	if len(whoisCache) >= whoisCacheMax {
		whoisCache = map[string]whoisCacheEntry{}
	}
	whoisCache[domain] = whoisCacheEntry{time.Now(), result}
	whoisCacheMu.Unlock()
	return result
}
