package ipinfo

import (
	"strings"
	"testing"

	"shortener/internal/whois"
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
		if got := NormalizeQuery(in); got != want {
			t.Errorf("NormalizeQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckPublic(t *testing.T) {
	for ip, want := range map[string]string{"8.8.8.8": "", "192.168.1.1": "err_private", "127.0.0.1": "err_private", "240.0.0.1": "err_reserved", "224.0.0.1": "err_reserved"} {
		got := ""
		if err := checkPublic(ip); err != nil {
			got = err.(*LookupErr).key
		}
		if got != want {
			t.Errorf("checkPublic(%s) = %q, want %q", ip, got, want)
		}
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
	if got := FormatPlace(&Info{City: "Moscow", RegionName: "Moscow", Country: "Russia"}, true); got != "Moscow, Russia" {
		t.Error(got)
	}
	if got := formatAS(&Info{As: "AS15169 Google LLC", Isp: "Google LLC"}); got != "AS15169" {
		t.Error(got)
	}
	if got := formatAS(&Info{As: "AS15169 Google LLC", AsName: "GOOGLE"}); got != "AS15169 (GOOGLE)" {
		t.Error(got)
	}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	if got := formatDate(now.AddDate(-29, 0, 0), "ru", now); !strings.Contains(got, "29 лет назад") {
		t.Error(got)
	}
	w := &whois.Info{Domain: "example.com", Expires: now.AddDate(0, 0, 10)}
	if rows := formatWhois(w, "example.com", "en", now); len(rows) != 1 || !strings.HasPrefix(rows[0], warnIcon) {
		t.Errorf("%v", rows)
	}
	if got := formatTimezone("Europe/Moscow", "en", now); got != "Europe/Moscow (UTC+3), now 15:00" {
		t.Error(got)
	}
}
