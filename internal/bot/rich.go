package bot

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"shortener/internal/i18n"
	"shortener/internal/ipinfo"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// the rich message needs a real emoji inside <tg-emoji>, "•" is rejected (RICH_MESSAGE_EMOJI_INVALID)
var iconFallback = map[string]string{"5222255236332168190": "🔗", "5222258908529207849": "🌐", "5292081728003808363": "🦖"}

// richButton renders one button as an in-text <tg-button> (Bot API 10.3 rich messages).
func richButton(b button) string {
	attr := ""
	switch {
	case b.WebApp != nil:
		attr = `type="web_app" url="` + html.EscapeString(b.WebApp.URL) + `"`
	case b.Copy != nil:
		attr = `type="copy_text" text="` + html.EscapeString(b.Copy.Text) + `"`
	case b.URL != "":
		attr = `type="url" url="` + html.EscapeString(b.URL) + `"`
	default:
		attr = `type="callback_data" data="` + html.EscapeString(b.Data) + `"`
	}
	if b.Style != "" {
		attr += ` style="` + b.Style + `"`
	}
	label := html.EscapeString(b.Text)
	if b.Icon != "" {
		label = `<tg-emoji emoji-id="` + b.Icon + `">` + iconFallback[b.Icon] + `</tg-emoji> ` + label
	}
	return "<tg-button " + attr + ">" + label + "</tg-button>"
}

// sendCard sends text with buttons inside the message (sendRichMessage).
// shortcut: falls back to a plain message with an inline keyboard if the rich method is rejected.
func sendCard(bot *tgbotapi.BotAPI, chatID int64, replyTo int, text string, kb keyboard) {
	body := strings.ReplaceAll(text, "\n", "<br>")
	for _, row := range kb.Rows {
		parts := make([]string, len(row))
		for i, b := range row {
			parts[i] = richButton(b)
		}
		body += "<tg-button-row>" + strings.Join(parts, "") + "</tg-button-row>"
	}
	rich, _ := json.Marshal(map[string]string{"html": body})
	params := tgbotapi.Params{"chat_id": strconv.FormatInt(chatID, 10), "rich_message": string(rich)}
	if replyTo != 0 {
		params["reply_parameters"] = fmt.Sprintf(`{"message_id":%d}`, replyTo)
	}
	if _, err := bot.MakeRequest("sendRichMessage", params); err == nil {
		return
	} else {
		log.Printf("rich: %v", err)
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyToMessageID = replyTo
	if len(kb.Rows) > 0 {
		msg.ReplyMarkup = kb
	}
	if _, err := bot.Send(msg); err != nil {
		log.Printf("send: %v", err)
	}
}

func lookupKeyboard(lang string, info *ipinfo.Info) keyboard {
	row := []button{{Text: i18n.Tr(lang, "btn_copy"), Copy: &copyText{info.Query}}}
	if info.HasCoords() {
		row = append(row, button{Text: i18n.Tr(lang, "btn_map"), Style: "primary",
			URL: fmt.Sprintf("https://maps.google.com/?q=%f,%f", *info.Lat, *info.Lon)})
	}
	return keyboard{[][]button{row}}
}
