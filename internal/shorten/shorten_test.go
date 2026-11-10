package shorten

import "testing"

func TestShorten(t *testing.T) {
	u := Shorten("example.com")
	store.RLock()
	got := store.urls[u[len(domain)+1:]]
	store.RUnlock()
	if got != "http://example.com" {
		t.Fatalf("got %q", got)
	}
}
