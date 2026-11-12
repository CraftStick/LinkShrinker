package bot

import (
	"strings"
	"testing"
)

func TestRichButton(t *testing.T) {
	cases := map[string]button{
		`type="web_app" url="https://h/p"`: {Text: "x", WebApp: &webApp{"https://h/p"}},
		`type="callback_data" data="m:ip"`: {Text: "x", Data: "m:ip"},
		`style="primary"`:                  {Text: "x", URL: "https://h", Style: "primary"},
	}
	for want, b := range cases {
		if got := richButton(b); !strings.Contains(got, want) {
			t.Errorf("%q not in %q", want, got)
		}
	}
	// every icon used on the menu needs a real fallback emoji, or Telegram rejects the whole message
	for _, row := range menuKeyboard("ru").Rows {
		for _, b := range row {
			if b.Icon != "" && iconFallback[b.Icon] == "" {
				t.Errorf("no fallback emoji for icon %s", b.Icon)
			}
		}
	}
}
