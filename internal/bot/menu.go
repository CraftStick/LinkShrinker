package bot

import (
	"encoding/json"
	"html"
	"log"
	"shortener/internal/game"
	"shortener/internal/i18n"
	"shortener/internal/ipinfo"
	"shortener/internal/shorten"
	"strconv"
	"strings"
	"sync"

	"github.com/skip2/go-qrcode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	modeMu sync.Mutex
	modes  = map[int64]string{}
)

func setMode(chatID int64, mode string) {
	modeMu.Lock()
	defer modeMu.Unlock()
	if mode == "" {
		delete(modes, chatID)
	} else {
		modes[chatID] = mode
	}
}

func takeMode(chatID int64) string {
	modeMu.Lock()
	defer modeMu.Unlock()
	mode := modes[chatID]
	delete(modes, chatID)
	return mode
}

type button struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
	Icon string `json:"icon_custom_emoji_id,omitempty"`
}

type keyboard struct {
	Rows [][]button `json:"inline_keyboard"`
}

func menuKeyboard(lang string) keyboard {
	return keyboard{[][]button{
		{
			{i18n.Tr(lang, "btn_short"), "m:short", "5222255236332168190"},
			{i18n.Tr(lang, "btn_ip"), "m:ip", "5222258908529207849"},
		},
		{{i18n.Tr(lang, "btn_game"), "m:game", "5292081728003808363"}},
	}}
}

func cancelKeyboard(lang string) keyboard {
	return keyboard{[][]button{{{i18n.Tr(lang, "btn_cancel"), "m:cancel", ""}}}}
}

func sendWithMarkup(bot *tgbotapi.BotAPI, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = markup
	bot.Send(msg)
}

func editMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, text string, markup keyboard) {
	kb, _ := json.Marshal(markup)
	bot.MakeRequest("editMessageText", tgbotapi.Params{
		"chat_id":                  strconv.FormatInt(msg.Chat.ID, 10),
		"message_id":               strconv.Itoa(msg.MessageID),
		"text":                     text,
		"parse_mode":               tgbotapi.ModeHTML,
		"disable_web_page_preview": "true",
		"reply_markup":             string(kb),
	})
}

func handleCallback(bot *tgbotapi.BotAPI, q *tgbotapi.CallbackQuery) {
	bot.Request(tgbotapi.NewCallback(q.ID, ""))
	if q.Message == nil {
		return
	}
	chatID, lang := q.Message.Chat.ID, i18n.LangOf(q.From.LanguageCode)
	if url, ok := strings.CutPrefix(q.Data, "qr:"); ok {
		png, err := qrcode.Encode(url, qrcode.Medium, 512)
		if err != nil {
			log.Printf("qr: %v", err)
			return
		}
		if _, err := bot.Send(tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "qr.png", Bytes: png})); err != nil {
			log.Printf("qr send: %v", err)
		}
		return
	}
	switch q.Data {
	case "m:short":
		if !canShorten(q.From.ID, q.Message.Chat) {
			sendHTML(bot, chatID, 0, i18n.Tr(lang, "short_denied"))
			return
		}
		setMode(chatID, "short")
		editMenu(bot, q.Message, i18n.Tr(lang, "ask_url"), cancelKeyboard(lang))
	case "m:ip":
		setMode(chatID, "ip")
		editMenu(bot, q.Message, i18n.Tr(lang, "ask_ip"), cancelKeyboard(lang))
	case "m:game":
		if q.Message.Chat.Type != "private" {
			sendHTML(bot, chatID, 0, i18n.Tr(lang, "game_private"))
			return
		}
		kb, _ := json.Marshal(tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Tr(lang, "btn_cancel"), "g:back"))))
		if err := game.Send(bot, chatID, string(kb)); err != nil {
			log.Printf("game: %v", err)
			return
		}
		bot.Request(tgbotapi.NewDeleteMessage(chatID, q.Message.MessageID))
	case "g:back":
		sendWithMarkup(bot, chatID, startText(bot, q.From, lang), menuKeyboard(lang))
		bot.Request(tgbotapi.NewDeleteMessage(chatID, q.Message.MessageID))
	case "m:cancel":
		setMode(chatID, "")
		editMenu(bot, q.Message, startText(bot, q.From, lang), menuKeyboard(lang))
	}
}

func canShorten(userID int64, chat *tgbotapi.Chat) bool {
	return allowedUsers[userID] && chat.Type == "private"
}

func handleText(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	lang := i18n.LangOf(m.From.LanguageCode)
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(m.Text), "@"+bot.Self.UserName))
	mode := takeMode(m.Chat.ID)
	if mode == "" {
		switch {
		case text == "":
		case shorten.HasScheme(text):
			mode = "short"
		case ipinfo.NormalizeQuery(text) != "":
			mode = "ip"
		}
	}
	switch mode {
	case "short":
		if !canShorten(m.From.ID, m.Chat) {
			sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "short_denied"))
			return
		}
		short := shorten.Shorten(text)
		msg := tgbotapi.NewMessage(m.Chat.ID, i18n.Tr(lang, "short_result", "url", short))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.DisableWebPagePreview = true
		msg.ReplyToMessageID = m.MessageID
		msg.ReplyMarkup = keyboard{[][]button{{{i18n.Tr(lang, "btn_qr"), "qr:" + short, ""}}}}
		if _, err := bot.Send(msg); err != nil {
			log.Printf("send: %v", err)
		}
	case "ip":
		replyLookup(bot, m, lang, text)
	default:
		sendWithMarkup(bot, m.Chat.ID, i18n.Tr(lang, "menu"), menuKeyboard(lang))
	}
}

func startText(bot *tgbotapi.BotAPI, u *tgbotapi.User, lang string) string {
	return i18n.Tr(lang, "start", "name", html.EscapeString(u.FirstName), "bot", botUsername(bot))
}
