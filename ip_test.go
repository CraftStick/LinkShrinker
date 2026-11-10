package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeQuery(t *testing.T) {
	for in, want := range map[string]string{
		"8.8.8.8":                  "8.8.8.8",
		"8.8.8.8:53":               "8.8.8.8",
		"https://Example.com/page": "example.com",
		"https://example.com:8443": "example.com",
		"example.com.":             "example.com",
		"[2001:db8::1]:443":        "2001:db8::1",
		"2001:4860:4860::8888":     "2001:4860:4860::8888",
		"пример.рф":                "xn--e1afmkfd.xn--p1ai",
		"not a domain":             "",
		"999.1.1.1":                "",
		"localhost":                "",
		"http://[::1":              "",
		"":                         "",
	} {
		if got := normalizeQuery(in); got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckPublic(t *testing.T) {
	for ip, want := range map[string]string{"8.8.8.8": "", "192.168.1.1": "err_private", "127.0.0.1": "err_private", "240.0.0.1": "err_reserved", "224.0.0.1": "err_reserved"} {
		got := ""
		if err := checkPublic(ip); err != nil {
			got = err.(*lookupErr).key
		}
		if got != want {
			t.Errorf("checkPublic(%s) = %q, want %q", ip, got, want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	for _, c := range []struct {
		lang string
		days int
		want string
	}{{"en", 1, "1 day"}, {"en", 400, "1 year"}, {"en", 800, "2 years"}, {"en", 95, "3 months"},
		{"ru", 10600, "29 лет"}, {"ru", 365, "1 год"}, {"ru", 800, "2 года"}, {"ru", 3, "3 дня"}, {"ru", -12, "12 дней"}} {
		if got := formatDuration(c.lang, c.days); got != c.want {
			t.Errorf("formatDuration(%s,%d) = %q, want %q", c.lang, c.days, got, c.want)
		}
	}
}

func TestParseWhois(t *testing.T) {
	flat := "Domain Name: EXAMPLE.COM\nRegistrar: Example Registrar, Inc.\nCreation Date: 1995-08-14T04:00:00Z\nRegistry Expiry Date: 2030-08-13T04:00:00Z\n"
	w := parseWhois(flat, "example.com")
	if w == nil || w.Registrar != "Example Registrar, Inc." || w.Created.Year() != 1995 || w.Expires.Year() != 2030 {
		t.Fatalf("flat: %+v", w)
	}
	block := "Domain: example.eu\nRegistrar:\n    Name: Block Registrar\n    Website: x\nregistered: 15.09.1997\npaid-till: 2028.09.14\n"
	w = parseWhois(block, "example.eu")
	if w == nil || w.Registrar != "Block Registrar" || w.Created.Year() != 1997 || w.Expires.Month() != 9 {
		t.Fatalf("block: %+v", w)
	}
	if parseWhois("No entries found\n", "x.com") != nil {
		t.Error("expected nil for empty response")
	}
}

func TestParseRDAP(t *testing.T) {
	data := map[string]any{
		"ldhName": "EXAMPLE.COM",
		"events": []any{
			map[string]any{"eventAction": "registration", "eventDate": "1995-08-14T04:00:00Z"},
			map[string]any{"eventAction": "expiration", "eventDate": "2030-08-13T04:00:00Z"},
		},
		"entities": []any{map[string]any{"roles": []any{"registrar"},
			"vcardArray": []any{"vcard", []any{[]any{"fn", map[string]any{}, "text", "Reg Inc"}}}}},
	}
	w := parseRDAP(data, "example.com")
	if w.Domain != "example.com" || w.Registrar != "Reg Inc" || w.Created.Year() != 1995 || w.Expires.Year() != 2030 {
		t.Fatalf("%+v", w)
	}
}

func TestQuota(t *testing.T) {
	var q quota
	if !q.take(10) {
		t.Fatal("unknown quota must allow")
	}
	q.update(11, time.Minute)
	if !q.take(10) || q.take(10) || !q.take(0) {
		t.Error("reserve handling is wrong")
	}
	q.update(0, -time.Second)
	if !q.take(0) {
		t.Error("expired window must allow")
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := formatUTCOffset(3 * 3600); got != "UTC+3" {
		t.Error(got)
	}
	if got := formatUTCOffset(-(5*3600 + 1800)); got != "UTC−5:30" {
		t.Error(got)
	}
	if got := formatPlace(&ipInfo{City: "Moscow", RegionName: "Moscow", Country: "Russia"}, true); got != "Moscow, Russia" {
		t.Error(got)
	}
	if got := formatAS(&ipInfo{As: "AS15169 Google LLC", Isp: "Google LLC"}); got != "AS15169" {
		t.Error(got)
	}
	if got := formatAS(&ipInfo{As: "AS15169 Google LLC", AsName: "GOOGLE"}); got != "AS15169 (GOOGLE)" {
		t.Error(got)
	}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	if got := formatDate(now.AddDate(-29, 0, 0), "ru", now); !strings.Contains(got, "29 лет назад") {
		t.Error(got)
	}
	w := &domainInfo{Domain: "example.com", Expires: now.AddDate(0, 0, 10)}
	if rows := formatWhois(w, "example.com", "en", now); len(rows) != 1 || !strings.HasPrefix(rows[0], "⚠️") {
		t.Errorf("%v", rows)
	}
	if got := formatTimezone("Europe/Moscow", "en", now); got != "Europe/Moscow (UTC+3), now 15:00" {
		t.Error(got)
	}
}
