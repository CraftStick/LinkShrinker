package bot

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"shortener/internal/i18n"
	"shortener/internal/ipinfo"
	"shortener/internal/trace"
)

// The widget has no network: it only draws chat_data (<=1024 bytes) that the bot rewrites per hop.
// Compressed size must stay <= 4096 bytes; TestTraceWidgetFits checks it.
const traceWidget = `<style>.r{display:flex;align-items:center;gap:8px;padding:2px 12px;font-size:14px}.n{width:20px;text-align:right;opacity:.5}.w{flex:1}.b{height:6px;border-radius:3px;background:var(--tg-theme-button-color)}.s{opacity:.6;width:52px;text-align:right}</style><div id=t style="padding:8px 0"></div><script>
var W=TelegramWidget,T=document.getElementById('t'),ru=(navigator.language||'').slice(0,2)=='ru';
function e(s){return String(s).replace(/[&<>"]/g,function(c){return'&#'+c.charCodeAt(0)+';'})}
function fl(c){return c.length==2?String.fromCodePoint.apply(null,c.toUpperCase().split('').map(function(x){return 127397+x.charCodeAt(0)})):'·'}
function R(){var d={};try{d=JSON.parse(W.chatData)||{}}catch(x){}
var n=d.n||[],mx=Math.max.apply(null,[100].concat(n.map(function(x){return x[1]}))),cc={};
var h='<div class=r><b>'+e(d.h||'')+'</b></div>';
n.forEach(function(x,i){if(x[0])cc[x[0]]=1;h+='<div class=r><span class=n>'+(i+1)+'</span><span>'+fl(x[0])+'</span><div class=w>'+(x[1]>=0?'<div class=b style="width:'+Math.max(2,x[1]/mx*100)+'%"></div>':'')+'</div><span class=s>'+(x[1]>=0?x[1]+' ms':'*')+'</span></div>'});
h+='<div class=r style="opacity:.7">'+(d.d?(ru?n.length+' узлов, стран: ':n.length+' hops, countries: ')+Object.keys(cc).length:'…')+'</div>';
T.innerHTML=h}
W.onEvent('chatDataChanged',R);R();
</script>`

type traceRow [2]any // [country code, ms or -1]

func traceData(host string, rows []traceRow, done bool) string {
	d := 0
	if done {
		d = 1
	}
	b, _ := json.Marshal(map[string]any{"h": host, "d": d, "n": rows})
	return string(b)
}

func handleTrace(bot *tgbotapi.BotAPI, m *tgbotapi.Message, lang, text string) {
	host := ipinfo.NormalizeQuery(text)
	if host == "" || strings.HasPrefix(host, "-") {
		sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "err_invalid"))
		return
	}
	chatID := strconv.FormatInt(m.Chat.ID, 10)
	resp, err := bot.MakeRequest("sendWidget", tgbotapi.Params{
		"chat_id": chatID, "html": traceWidget, "chat_data": traceData(host, nil, false),
	})
	var msg struct {
		ID int `json:"message_id"`
	}
	if err == nil {
		err = json.Unmarshal(resp.Result, &msg)
	}
	if err != nil {
		log.Printf("sendWidget: %v", err)
		sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "trace_fail"))
		return
	}
	edit := func(rows []traceRow, done bool) {
		if _, err := bot.MakeRequest("editMessageWidgetData", tgbotapi.Params{
			"chat_id": chatID, "message_id": strconv.Itoa(msg.ID), "chat_data": traceData(host, rows, done),
		}); err != nil {
			log.Printf("editMessageWidgetData: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var rows []traceRow
	err = trace.Run(ctx, host, func(h trace.Hop) {
		row := traceRow{"", -1}
		if h.IP != "" {
			row[1] = int(h.MS + 0.5)
			// private and unknown nodes fail the lookup and stay without a flag
			if info, _, err := ipinfo.Lookup(h.IP, "en", ipinfo.APIReserve); err == nil {
				row[0] = info.CountryCode
			}
		}
		rows = append(rows, row)
		edit(rows, false)
	})
	if err != nil {
		log.Printf("trace %s: %v", host, err)
	}
	edit(rows, true)
}
