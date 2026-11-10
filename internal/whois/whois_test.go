package whois

import (
	"testing"
)

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
