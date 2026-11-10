package bot

import (
	"encoding/json"
	"log"
	"shortener/internal/game"
	"shortener/internal/i18n"
	"shortener/internal/ipinfo"
	"shortener/internal/shorten"
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
			tgbotapi.NewInlineKeyboardButtonData(i18n.Tr(lang, "btn_short"), "m:short"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.Tr(lang, "btn_ip"), "m:ip"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Tr(lang, "btn_game"), "m:game"),
		),
	)
}

func cancelKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.Tr(lang, "btn_cancel"), "m:cancel"),
	))
}

func sendWithMarkup(bot *tgbotapi.BotAPI, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = markup
	bot.Send(msg)
}

func editMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, text string, markup tgbotapi.InlineKeyboardMarkup) {
	e := tgbotapi.NewEditMessageTextAndMarkup(msg.Chat.ID, msg.MessageID, text, markup)
	e.ParseMode = tgbotapi.ModeHTML
	e.DisableWebPagePreview = true
	bot.Request(e)
}

func handleCallback(bot *tgbotapi.BotAPI, q *tgbotapi.CallbackQuery) {
	bot.Request(tgbotapi.NewCallback(q.ID, ""))
	if q.Message == nil {
		return
	}
	chatID, lang := q.Message.Chat.ID, i18n.LangOf(q.From.LanguageCode)
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
		sendWithMarkup(bot, chatID, i18n.Tr(lang, "menu"), menuKeyboard(lang))
		bot.Request(tgbotapi.NewDeleteMessage(chatID, q.Message.MessageID))
	case "m:cancel":
		setMode(chatID, "")
		editMenu(bot, q.Message, i18n.Tr(lang, "menu"), menuKeyboard(lang))
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
		sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "short_result", "url", shorten.Shorten(text)))
	case "ip":
		replyLookup(bot, m, lang, text)
	default:
		sendWithMarkup(bot, m.Chat.ID, i18n.Tr(lang, "menu"), menuKeyboard(lang))
	}
}
