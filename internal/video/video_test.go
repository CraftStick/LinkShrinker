package video

import "testing"

func TestSupported(t *testing.T) {
	for u, want := range map[string]bool{
		"https://vm.tiktok.com/ZM123/":      true,
		"https://www.instagram.com/reel/x/": true,
		"https://youtu.be/abc":              true,
		"https://x.com/a/status/1":          true,
		"https://nottiktok.com/x":           false,
		"https://example.com/a.mp4":         false,
		"tiktok.com/x":                      false,
		"https://tiktok.com.evil.io/x":      false,
	} {
		if Supported(u) != want {
			t.Errorf("%s: want %v", u, want)
		}
	}
}
