package i18n

import "testing"

func TestFormatDuration(t *testing.T) {
	for _, c := range []struct {
		lang string
		days int
		want string
	}{{"en", 1, "1 day"}, {"en", 400, "1 year"}, {"en", 800, "2 years"}, {"en", 95, "3 months"},
		{"ru", 10600, "29 лет"}, {"ru", 365, "1 год"}, {"ru", 800, "2 года"}, {"ru", 3, "3 дня"}, {"ru", -12, "12 дней"}} {
		if got := FormatDuration(c.lang, c.days); got != c.want {
			t.Errorf("FormatDuration(%s,%d) = %q, want %q", c.lang, c.days, got, c.want)
		}
	}
}
