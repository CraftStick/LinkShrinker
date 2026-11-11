package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"shortener/internal/i18n"
)

var (
	botAPI atomic.Pointer[tgbotapi.BotAPI]
	token  atomic.Value
)

const myipPage = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width">
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<body style="font:16px sans-serif;text-align:center;padding:2em">...
<script>
const w=Telegram.WebApp;w.ready();
fetch("/myip/check",{method:"POST",body:w.initData}).finally(()=>w.close());
</script>`

// MyIPPage serves the mini-app page opened by the "My IP" button.
func MyIPPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(myipPage))
}

// validInitData checks the Telegram WebApp signature and returns the user id and language.
func validInitData(initData, tok string) (int64, string, bool) {
	vals, err := url.ParseQuery(initData)
	if err != nil || tok == "" {
		return 0, "", false
	}
	hash := vals.Get("hash")
	vals.Del("hash")
	var parts []string
	for k := range vals {
		parts = append(parts, k+"="+vals.Get(k))
	}
	sort.Strings(parts)
	key := hmac.New(sha256.New, []byte("WebAppData"))
	key.Write([]byte(tok))
	mac := hmac.New(sha256.New, key.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(hash)) {
		return 0, "", false
	}
	var ts int64
	if _, err := fmt.Sscan(vals.Get("auth_date"), &ts); err != nil || time.Since(time.Unix(ts, 0)) > time.Hour {
		return 0, "", false
	}
	var u struct {
		ID   int64  `json:"id"`
		Lang string `json:"language_code"`
	}
	if json.Unmarshal([]byte(vals.Get("user")), &u) != nil || u.ID == 0 {
		return 0, "", false
	}
	return u.ID, u.Lang, true
}

// MyIPCheck verifies the caller and sends them a lookup card for the IP this request came from.
func MyIPCheck(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	tok, _ := token.Load().(string)
	id, lang, ok := validInitData(string(b), tok)
	bot := botAPI.Load()
	if !ok || bot == nil || !allowedUsers[id] {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ip := strings.TrimSpace(strings.Split(r.Header.Get("X-Real-IP"), ",")[0])
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	if net.ParseIP(ip) == nil {
		http.Error(w, "bad ip", http.StatusBadRequest)
		return
	}
	replyLookup(bot, &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: id}}, i18n.LangOf(lang), ip)
}
