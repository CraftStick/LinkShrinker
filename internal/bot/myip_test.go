package bot

import "testing"

func TestIPRoundTrip(t *testing.T) {
	for _, ip := range []string{"8.8.8.8", "2606:4700::1111", "::1"} {
		if got := decodeIP(encodeIP(ip)); got != ip {
			t.Errorf("%s -> %q", ip, got)
		}
	}
	if decodeIP("not_an_ip") != "" {
		t.Error("garbage accepted")
	}
}
