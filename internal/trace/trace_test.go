package trace

import "testing"

func TestParseHop(t *testing.T) {
	cases := []struct {
		in   string
		want Hop
		ok   bool
	}{
		{" 1  192.168.1.1  1.234 ms", Hop{1, "192.168.1.1", 1.234}, true},
		{" 3  *", Hop{N: 3}, true},
		{"traceroute to example.com (93.184.216.34), 20 hops max", Hop{}, false},
		{"", Hop{}, false},
	}
	for _, c := range cases {
		got, ok := ParseHop(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("%q: got %+v %v", c.in, got, ok)
		}
	}
}
