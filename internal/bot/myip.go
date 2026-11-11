package bot

import (
	"net"
	"net/http"
	"strings"
	"sync/atomic"
)

var botName atomic.Value

func encodeIP(ip string) string {
	return strings.NewReplacer(":", "-", ".", "_").Replace(ip)
}

func decodeIP(s string) string {
	ip := strings.NewReplacer("-", ":", "_", ".").Replace(s)
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// MyIPHandler sends the visitor back to the bot with their IP in the start parameter; the bot cannot see it itself.
func MyIPHandler(w http.ResponseWriter, r *http.Request) {
	name, _ := botName.Load().(string)
	ip := strings.TrimSpace(strings.Split(r.Header.Get("X-Real-IP"), ",")[0])
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	if name == "" || net.ParseIP(ip) == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "https://t.me/"+name+"?start=ip_"+encodeIP(ip), http.StatusFound)
}
