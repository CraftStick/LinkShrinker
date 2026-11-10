package main

import (
	"html"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	inlineCacheTime = 300
	inlineDebounce  = 600 * time.Millisecond
	pinThumbnailURL = "https://cdn.jsdelivr.net/gh/CraftStick/IPBot-telegram@main/docs/pin.jpg"
	pinThumbnailPx  = 136
)

func sendHTML(bot *tgbotapi.BotAPI, chatID int64, replyTo int, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyToMessageID = replyTo
	if _, err := bot.Send(msg); err != nil {
		log.Printf("send: %v", err)
	}
}

func botUsername(bot *tgbotapi.BotAPI) string { return bot.Self.UserName }

func handleStart(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	if m.CommandArguments() == "help" {
		handleHelp(bot, m)
		return
	}
	lang := langOf(m.From.LanguageCode)
	text := tr(lang, "start", "name", html.EscapeString(m.From.FirstName), "bot", botUsername(bot))
	sendHTML(bot, m.Chat.ID, 0, text)
}

func handleHelp(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	sendHTML(bot, m.Chat.ID, 0, tr(langOf(m.From.LanguageCode), "help", "bot", botUsername(bot)))
}

func handleIP(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	lang := langOf(m.From.LanguageCode)
	args := strings.TrimSpace(m.CommandArguments())
	if args == "" {
		sendHTML(bot, m.Chat.ID, m.MessageID, tr(lang, "need_address"))
		return
	}
	query := normalizeQuery(args)
	if query == "" {
		sendHTML(bot, m.Chat.ID, m.MessageID, tr(lang, "invalid_input"))
		return
	}
	info, whois, err := lookup(query, lang, 0)
	if err != nil {
		sendHTML(bot, m.Chat.ID, m.MessageID, "❌ "+err.(*lookupErr).text(lang))
		return
	}
	card := formatIPInfo(info, query, lang, whois)
	if info.hasCoords() {
		title, address := venueTexts(info)
		venue := tgbotapi.NewVenue(m.Chat.ID, title, address, *info.Lat, *info.Lon)
		venue.ReplyToMessageID = m.MessageID
		if _, err := bot.Send(venue); err != nil {
			log.Printf("venue for %s: %v", query, err)
		} else {
			sendHTML(bot, m.Chat.ID, 0, card)
			return
		}
	}
	sendHTML(bot, m.Chat.ID, m.MessageID, card)
}

// ---------- inline mode ----------

var (
	latestInlineMu sync.Mutex
	latestInline   = map[int64]string{}
)

// typingPaused waits a moment; false if the user typed on and a newer query replaced this one.
func typingPaused(q *tgbotapi.InlineQuery) bool {
	latestInlineMu.Lock()
	latestInline[q.From.ID] = q.ID
	latestInlineMu.Unlock()
	time.Sleep(inlineDebounce)
	latestInlineMu.Lock()
	defer latestInlineMu.Unlock()
	if latestInline[q.From.ID] != q.ID {
		return false
	}
	delete(latestInline, q.From.ID)
	return true
}

func handleInline(bot *tgbotapi.BotAPI, q *tgbotapi.InlineQuery) {
	lang := langOf(q.From.LanguageCode)
	var results []any
	if query := normalizeQuery(q.Query); query != "" {
		if !typingPaused(q) {
			return
		}
		if info, whois, err := lookup(query, lang, apiReserve); err == nil {
			var parts []string
			for _, p := range []string{formatPlace(info, false), info.Isp} {
				if p != "" {
					parts = append(parts, p)
				}
			}
			results = append(results, map[string]any{
				"type": "article", "id": "info",
				"title":       flag(info.CountryCode) + " " + info.Query,
				"description": strings.Join(parts, " · "),
				"input_message_content": tgbotapi.InputTextMessageContent{
					Text: formatIPInfo(info, query, lang, whois), ParseMode: tgbotapi.ModeHTML, DisableWebPagePreview: true,
				},
			})
			if info.hasCoords() {
				title, address := venueTexts(info)
				results = append(results, map[string]any{
					"type": "venue", "id": "location", "title": title, "address": address,
					"latitude": *info.Lat, "longitude": *info.Lon,
					"thumbnail_url": pinThumbnailURL, "thumbnail_width": pinThumbnailPx, "thumbnail_height": pinThumbnailPx,
				})
			}
		}
	}
	if results == nil {
		results = []any{}
	}
	params := tgbotapi.Params{}
	params["inline_query_id"] = q.ID
	params["cache_time"] = "300"
	params["is_personal"] = "true"
	if err := params.AddInterface("results", results); err != nil {
		log.Printf("inline: %v", err)
		return
	}
	if len(results) == 0 {
		params.AddInterface("button", map[string]string{"text": tr(lang, "inline_hint"), "start_parameter": "help"})
	}
	if _, err := bot.MakeRequest("answerInlineQuery", params); err != nil {
		log.Printf("answerInlineQuery: %v", err)
	}
}

func setupCommands(bot *tgbotapi.BotAPI) {
	for _, lang := range languages {
		cmds := []tgbotapi.BotCommand{
			{Command: "ip", Description: tr(lang, "cmd_ip")},
			{Command: "help", Description: tr(lang, "cmd_help")},
			{Command: "start", Description: tr(lang, "cmd_start")},
		}
		code := lang
		if lang == defaultLang {
			code = ""
		}
		cfg := tgbotapi.SetMyCommandsConfig{Commands: cmds, LanguageCode: code}
		if _, err := bot.Request(cfg); err != nil {
			log.Printf("setMyCommands %s: %v", lang, err)
		}
	}
}
