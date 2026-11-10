package main

import (
	"fmt"
	"strings"
	"sync"

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

func menuKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(tr(lang, "btn_short"), "m:short"),
			tgbotapi.NewInlineKeyboardButtonData(tr(lang, "btn_ip"), "m:ip"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(tr(lang, "btn_help"), "m:help"),
		),
	)
}

func cancelKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(tr(lang, "btn_cancel"), "m:cancel"),
	))
}

func sendWithMarkup(bot *tgbotapi.BotAPI, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = markup
	bot.Send(msg)
}

func handleCallback(bot *tgbotapi.BotAPI, q *tgbotapi.CallbackQuery) {
	bot.Request(tgbotapi.NewCallback(q.ID, ""))
	if q.Message == nil {
		return
	}
	chatID, lang := q.Message.Chat.ID, langOf(q.From.LanguageCode)
	switch q.Data {
	case "m:short":
		if !canShorten(q.From.ID, q.Message.Chat) {
			sendHTML(bot, chatID, 0, tr(lang, "short_denied"))
			return
		}
		setMode(chatID, "short")
		sendWithMarkup(bot, chatID, tr(lang, "ask_url"), cancelKeyboard(lang))
	case "m:ip":
		setMode(chatID, "ip")
		sendWithMarkup(bot, chatID, tr(lang, "ask_ip"), cancelKeyboard(lang))
	case "m:help":
		sendWithMarkup(bot, chatID, tr(lang, "help", "bot", botUsername(bot)), menuKeyboard(lang))
	case "m:cancel":
		setMode(chatID, "")
		sendHTML(bot, chatID, 0, tr(lang, "cancelled"))
	}
}

func canShorten(userID int64, chat *tgbotapi.Chat) bool {
	return allowedUsers[userID] && chat.Type == "private"
}

func hasScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func shortenURL(raw string) string {
	if !hasScheme(raw) {
		raw = "http://" + raw
	}
	id := generateID(6)
	store.Lock()
	store.urls[id] = raw
	store.Unlock()
	return fmt.Sprintf("%s/%s", domain, id)
}

func handleText(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	lang := langOf(m.From.LanguageCode)
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(m.Text), "@"+bot.Self.UserName))
	mode := takeMode(m.Chat.ID)
	if mode == "" {
		switch {
		case text == "":
		case hasScheme(text):
			mode = "short"
		case normalizeQuery(text) != "":
			mode = "ip"
		}
	}
	switch mode {
	case "short":
		if !canShorten(m.From.ID, m.Chat) {
			sendHTML(bot, m.Chat.ID, m.MessageID, tr(lang, "short_denied"))
			return
		}
		sendHTML(bot, m.Chat.ID, m.MessageID, tr(lang, "short_result", "url", shortenURL(text)))
	case "ip":
		replyLookup(bot, m, lang, text)
	default:
		sendWithMarkup(bot, m.Chat.ID, tr(lang, "menu"), menuKeyboard(lang))
	}
}
