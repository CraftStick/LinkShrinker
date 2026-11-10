package ipinfo

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"shortener/internal/i18n"
	"shortener/internal/whois"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
)

const (
	ipAPIURL       = "http://ip-api.com/json/%s"
	ipAPIFields    = "status,message,query,continent,country,countryCode,regionName,city,district,zip,timezone,isp,org,as,asname,reverse,lat,lon,mobile"
	fallbackURL    = "https://ipwho.is/%s"
	apiTimeout     = 5 * time.Second
	APIReserve     = 10
	fallbackRetry  = time.Hour
	cacheTTL       = 5 * time.Minute
	cacheMax       = 1000
	whoisWait      = 6 * time.Second
	expiryWarnDays = 30
	maxQueryLen    = 253
)

var (
	domainRe  = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	allDigits = regexp.MustCompile(`^[0-9.]+$`)
	apiClient = &http.Client{Timeout: apiTimeout}
	apiErrors = map[string]string{"private range": "err_private", "reserved range": "err_reserved", "invalid query": "err_invalid"}
)

type LookupErr struct {
	key     string
	message string
}

const (
	warnIcon = `<tg-emoji emoji-id="5276240711795107620">⚠️</tg-emoji> `
	pinIcon  = `<tg-emoji emoji-id="5803262460626472716">📍</tg-emoji>`
)

func (e *LookupErr) Error() string { return e.key }

func (e *LookupErr) Text(lang string) string {
	return i18n.Tr(lang, e.key, "message", html.EscapeString(e.message))
}

func (e *LookupErr) temporary() bool { return e.key == "err_rate_limit" || e.key == "err_unavailable" }

type Info struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	Query       string   `json:"query"`
	Continent   string   `json:"continent"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
	RegionName  string   `json:"regionName"`
	City        string   `json:"city"`
	District    string   `json:"district"`
	Zip         string   `json:"zip"`
	Timezone    string   `json:"timezone"`
	Isp         string   `json:"isp"`
	Org         string   `json:"org"`
	As          string   `json:"as"`
	AsName      string   `json:"asname"`
	Reverse     string   `json:"reverse"`
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
	Mobile      bool     `json:"mobile"`
}

func (i *Info) HasCoords() bool { return i.Lat != nil && i.Lon != nil }

func NormalizeQuery(text string) string {
	q := strings.TrimSpace(text)
	if q == "" {
		return ""
	}
	if strings.Contains(q, "://") {
		u, err := url.Parse(q)
		if err != nil {
			return ""
		}
		q = u.Host
	} else {
		q = strings.SplitN(q, "/", 2)[0]
	}
	switch {
	case strings.HasPrefix(q, "["):
		end := strings.Index(q, "]")
		if end < 0 {
			return ""
		}
		q = q[1:end]
	case strings.Count(q, ":") == 1:
		q = strings.Split(q, ":")[0]
	}
	q = strings.TrimRight(strings.TrimSpace(q), ".")
	if ip := net.ParseIP(q); ip != nil {
		return ip.String()
	}
	if q == "" || allDigits.MatchString(q) {
		return ""
	}
	ascii, err := idna.Lookup.ToASCII(q)
	if err != nil || len(ascii) > maxQueryLen || !domainRe.MatchString(ascii) {
		return ""
	}
	return ascii
}

func isIP(s string) bool { return net.ParseIP(s) != nil }

func displayDomain(d string) string {
	if u, err := idna.ToUnicode(d); err == nil {
		return u
	}
	return d
}

func checkPublic(ip string) error {
	p := net.ParseIP(ip)
	switch {
	case p == nil:
		return &LookupErr{key: "err_invalid"}
	case p.IsPrivate() || p.IsLoopback() || p.IsLinkLocalUnicast():
		return &LookupErr{key: "err_private"}
	case !p.IsGlobalUnicast() || inReserved(p):
		return &LookupErr{key: "err_reserved"}
	}
	return nil
}

var reservedNets = func() (n []*net.IPNet) {
	for _, c := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "100::/64"} {
		_, x, _ := net.ParseCIDR(c)
		n = append(n, x)
	}
	return
}()

func inReserved(ip net.IP) bool {
	for _, n := range reservedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func resolve(domain string) (string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil || len(ips) == 0 {
		return "", &LookupErr{key: "err_invalid"}
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return ips[0].String(), nil
}

type quota struct {
	mu        sync.Mutex
	known     bool
	remaining int
	resetAt   time.Time
}

func (q *quota) take(reserve int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.known || !time.Now().Before(q.resetAt) {
		return true
	}
	if q.remaining <= reserve {
		return false
	}
	q.remaining--
	return true
}

func (q *quota) update(remaining int, resetIn time.Duration) {
	q.mu.Lock()
	q.known, q.remaining, q.resetAt = true, remaining, time.Now().Add(resetIn)
	q.mu.Unlock()
}

var ipapiQuota, fallbackQuota quota

func ipapiLookup(query, lang string, reserve int) (*Info, error) {
	if !ipapiQuota.take(reserve) {
		return nil, &LookupErr{key: "err_rate_limit"}
	}
	u := fmt.Sprintf(ipAPIURL, url.PathEscape(query)) + "?fields=" + ipAPIFields
	if lang != i18n.DefaultLang {
		u += "&lang=" + lang
	}
	resp, err := apiClient.Get(u)
	if err != nil {
		log.Printf("ip-api request for %s failed: %v", query, err)
		return nil, &LookupErr{key: "err_unavailable"}
	}
	defer resp.Body.Close()
	rl, err1 := strconv.Atoi(resp.Header.Get("X-Rl"))
	ttl, err2 := strconv.Atoi(resp.Header.Get("X-Ttl"))
	if err1 == nil && err2 == nil {
		ipapiQuota.update(rl, time.Duration(ttl)*time.Second)
	} else if resp.StatusCode == http.StatusTooManyRequests {
		ipapiQuota.update(0, time.Minute)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &LookupErr{key: "err_rate_limit"}
	}
	var info Info
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&info) != nil {
		log.Printf("ip-api for %s: HTTP %d", query, resp.StatusCode)
		return nil, &LookupErr{key: "err_unavailable"}
	}
	if info.Status != "success" {
		if key, ok := apiErrors[info.Message]; ok {
			return nil, &LookupErr{key: key}
		}
		msg := info.Message
		if msg == "" {
			msg = "unknown error"
		}
		return nil, &LookupErr{key: "err_unknown", message: msg}
	}
	return &info, nil
}

func fallbackLookup(query, lang string) (*Info, error) {
	ip := query
	if !isIP(query) {
		var err error
		if ip, err = resolve(query); err != nil {
			return nil, err
		}
	}
	if err := checkPublic(ip); err != nil {
		return nil, err
	}
	if !fallbackQuota.take(0) {
		return nil, &LookupErr{key: "err_rate_limit"}
	}
	u := fmt.Sprintf(fallbackURL, url.PathEscape(ip))
	if lang != i18n.DefaultLang {
		u += "?lang=" + lang
	}
	resp, err := apiClient.Get(u)
	if err != nil {
		log.Printf("ipwho.is request for %s failed: %v", ip, err)
		return nil, &LookupErr{key: "err_unavailable"}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		retry := fallbackRetry
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
			retry = time.Duration(s) * time.Second
		}
		fallbackQuota.update(0, retry)
		return nil, &LookupErr{key: "err_rate_limit"}
	}
	var d struct {
		Success    bool                `json:"success"`
		Message    string              `json:"message"`
		IP         string              `json:"ip"`
		Continent  string              `json:"continent"`
		Country    string              `json:"country"`
		Code       string              `json:"country_code"`
		Region     string              `json:"region"`
		City       string              `json:"city"`
		Postal     string              `json:"postal"`
		Lat        *float64            `json:"latitude"`
		Lon        *float64            `json:"longitude"`
		Timezone   struct{ ID string } `json:"timezone"`
		Connection struct {
			ASN      int
			Org, Isp string
		} `json:"connection"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&d) != nil || !d.Success {
		log.Printf("ipwho.is lookup for %s failed: HTTP %d %s", ip, resp.StatusCode, d.Message)
		return nil, &LookupErr{key: "err_unavailable"}
	}
	info := &Info{Query: d.IP, Continent: d.Continent, Country: d.Country, CountryCode: d.Code, RegionName: d.Region,
		City: d.City, Zip: d.Postal, Timezone: d.Timezone.ID, Isp: d.Connection.Isp, Org: d.Connection.Org, Lat: d.Lat, Lon: d.Lon}
	if info.Query == "" {
		info.Query = ip
	}
	if d.Connection.ASN != 0 {
		info.As = "AS" + strconv.Itoa(d.Connection.ASN)
	}
	return info, nil
}

type cacheEntry struct {
	at   time.Time
	info *Info
}

var (
	cacheMu sync.Mutex
	cache   = map[[2]string]cacheEntry{}
)

func getIPInfo(query, lang string, reserve int) (*Info, error) {
	if isIP(query) {
		if err := checkPublic(query); err != nil {
			return nil, err
		}
	}
	key := [2]string{query, lang}
	cacheMu.Lock()
	c, ok := cache[key]
	cacheMu.Unlock()
	if ok && time.Since(c.at) < cacheTTL {
		return c.info, nil
	}
	info, err := ipapiLookup(query, lang, reserve)
	if err != nil {
		first := err.(*LookupErr)
		if !first.temporary() {
			return nil, err
		}
		var ferr error
		if info, ferr = fallbackLookup(query, lang); ferr != nil {
			if ferr.(*LookupErr).temporary() {
				return nil, first
			}
			return nil, ferr
		}
	}
	cacheMu.Lock()
	if len(cache) >= cacheMax {
		for k, v := range cache {
			if time.Since(v.at) >= cacheTTL {
				delete(cache, k)
			}
		}
		if len(cache) >= cacheMax {
			cache = map[[2]string]cacheEntry{}
		}
	}
	cache[key] = cacheEntry{time.Now(), info}
	cacheMu.Unlock()
	return info, nil
}

func Lookup(query, lang string, reserve int) (*Info, *whois.Info, error) {
	var whoisCh chan *whois.Info
	if !isIP(query) {
		whoisCh = make(chan *whois.Info, 1)
		go func() { whoisCh <- whois.Lookup(query) }()
	}
	info, err := getIPInfo(query, lang, reserve)
	if err != nil {
		return nil, nil, err
	}
	var wi *whois.Info
	if whoisCh != nil {
		select {
		case wi = <-whoisCh:
		case <-time.After(whoisWait):
			log.Printf("WHOIS for %s took too long", query)
		}
	}
	return info, wi, nil
}

func Flag(cc string) string {
	if len(cc) != 2 {
		return "🏳️"
	}
	cc = strings.ToUpper(cc)
	return string([]rune{rune(cc[0]) - 'A' + 0x1F1E6, rune(cc[1]) - 'A' + 0x1F1E6})
}

func FormatPlace(i *Info, withRegion bool) string {
	parts := []string{i.City, i.Country}
	if withRegion {
		parts = []string{i.District, i.City, i.RegionName, i.Country}
	}
	var out []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		dup := false
		for _, o := range out {
			dup = dup || o == p
		}
		if !dup {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

func formatUTCOffset(offset int) string {
	minutes := offset / 60
	sign := "+"
	if minutes < 0 {
		sign, minutes = "−", -minutes
	}
	s := fmt.Sprintf("UTC%s%d", sign, minutes/60)
	if minutes%60 != 0 {
		s += fmt.Sprintf(":%02d", minutes%60)
	}
	return s
}

func formatTimezone(name, lang string, now time.Time) string {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return html.EscapeString(name)
	}
	local := now.In(loc)
	_, off := local.Zone()
	return fmt.Sprintf("%s (%s), %s", html.EscapeString(name), formatUTCOffset(off), i18n.Tr(lang, "local_time", "time", local.Format("15:04")))
}

func formatAS(i *Info) string {
	number, name, _ := strings.Cut(i.As, " ")
	if i.AsName != "" {
		return number + " (" + i.AsName + ")"
	}
	if name != "" && !strings.EqualFold(name, i.Isp) && !strings.EqualFold(name, i.Org) {
		return i.As
	}
	return number
}

func dayOf(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysBetween(a, b time.Time) int { return int(dayOf(a).Sub(dayOf(b)).Hours() / 24) }

func formatDate(v time.Time, lang string, now time.Time) string {
	days := daysBetween(v, now)
	var rel string
	switch {
	case days == 0:
		rel = i18n.Tr(lang, "today")
	case days > 0:
		rel = i18n.Tr(lang, "in", "duration", i18n.FormatDuration(lang, days))
	default:
		rel = i18n.Tr(lang, "ago", "duration", i18n.FormatDuration(lang, days))
	}
	return fmt.Sprintf("%s (%s)", v.Format(i18n.Tr(lang, "date_format")), rel)
}

func row(lang, key, value string) string { return "<b>" + i18n.Tr(lang, key) + ":</b> " + value }

func formatWhois(w *whois.Info, query, lang string, now time.Time) []string {
	var rows []string
	if w.Domain != query {
		rows = append(rows, row(lang, "domain", "<code>"+html.EscapeString(displayDomain(w.Domain))+"</code>"))
	}
	if w.Registrar != "" {
		rows = append(rows, row(lang, "registrar", html.EscapeString(w.Registrar)))
	}
	if !w.Created.IsZero() {
		rows = append(rows, row(lang, "registered", formatDate(w.Created, lang, now)))
	}
	if !w.Expires.IsZero() {
		left := daysBetween(w.Expires, now)
		switch {
		case left < 0:
			rows = append(rows, warnIcon+row(lang, "expired", w.Expires.Format(i18n.Tr(lang, "date_format"))))
		case left <= expiryWarnDays:
			rows = append(rows, warnIcon+row(lang, "expires", formatDate(w.Expires, lang, now)))
		default:
			rows = append(rows, row(lang, "expires", formatDate(w.Expires, lang, now)))
		}
	}
	return rows
}

func Format(i *Info, query, lang string, w *whois.Info) string {
	e := html.EscapeString
	now := time.Now().UTC()
	var b strings.Builder
	if query != i.Query {
		fmt.Fprintf(&b, "%s <b>%s</b>\n<code>%s</code>\n", Flag(i.CountryCode), e(displayDomain(query)), e(i.Query))
	} else {
		fmt.Fprintf(&b, "%s <code>%s</code>\n", Flag(i.CountryCode), e(i.Query))
	}
	section := func(emoji, key string, rows []string) {
		if len(rows) > 0 {
			fmt.Fprintf(&b, "\n%s <b>%s</b>\n<blockquote>%s</blockquote>\n", emoji, i18n.Tr(lang, key), strings.Join(rows, "\n"))
		}
	}
	add := func(rows []string, key, value string) []string {
		if value == "" {
			return rows
		}
		return append(rows, row(lang, key, value))
	}
	var loc []string
	if place := FormatPlace(i, true); place != "" {
		loc = append(loc, e(place))
	}
	loc = add(loc, "continent", e(i.Continent))
	if i.Zip != "" {
		loc = add(loc, "zip", "<code>"+e(i.Zip)+"</code>")
	}
	if i.Timezone != "" {
		loc = add(loc, "timezone", formatTimezone(i.Timezone, lang, now))
	}
	if i.HasCoords() {
		loc = add(loc, "coords", fmt.Sprintf("<code>%v, %v</code>", *i.Lat, *i.Lon))
	}
	section(pinIcon, "section_location", loc)

	var net []string
	net = add(net, "isp", e(i.Isp))
	if !strings.EqualFold(i.Org, i.Isp) {
		net = add(net, "org", e(i.Org))
	}
	if i.As != "" {
		net = add(net, "as", e(formatAS(i)))
	}
	if i.Reverse != "" {
		net = add(net, "reverse", "<code>"+e(i.Reverse)+"</code>")
	}
	section("📡", "section_network", net)
	if w != nil {
		section("🌐", "section_whois", formatWhois(w, query, lang, now))
	}
	if i.Mobile {
		fmt.Fprintf(&b, "\n<blockquote>%s</blockquote>", i18n.Tr(lang, "tag_mobile"))
	}
	return b.String()
}

func VenueTexts(i *Info) (title, address string) {
	title = FormatPlace(i, true)
	if title == "" {
		title = i.Query
	}
	var parts []string
	for _, p := range []string{i.Query, i.Isp} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return title, strings.Join(parts, " · ")
}
