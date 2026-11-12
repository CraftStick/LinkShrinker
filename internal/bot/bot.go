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

// sendRichCard sends the lookup card as a Bot API 10.3 rich message with buttons inside the text.
// shortcut: falls back to a plain message if the rich method is rejected.
func sendRichCard(bot *tgbotapi.BotAPI, chatID int64, replyTo int, lang, card string, info *ipinfo.Info) {
	btns := `<tg-button type="copy_text" text="` + html.EscapeString(info.Query) + `">` + i18n.Tr(lang, "btn_copy") + `</tg-button>`
	if len(info.Query) <= 60 { // callback_data limit is 64 bytes
		btns += `<tg-button type="callback_data" data="qr:` + html.EscapeString(info.Query) + `">` + i18n.Tr(lang, "btn_qr") + `</tg-button>`
	}
	if info.HasCoords() {
		btns += fmt.Sprintf(`<tg-button type="url" style="primary" url="https://maps.google.com/?q=%f,%f">%s</tg-button>`, *info.Lat, *info.Lon, i18n.Tr(lang, "btn_map"))
	}
	rich, _ := json.Marshal(map[string]string{"html": strings.ReplaceAll(card, "\n", "<br>") + `<tg-button-row>` + btns + `</tg-button-row>`})
	params := tgbotapi.Params{"chat_id": strconv.FormatInt(chatID, 10), "rich_message": string(rich)}
	if replyTo != 0 {
		params["reply_parameters"] = fmt.Sprintf(`{"message_id":%d}`, replyTo)
	}
	if _, err := bot.MakeRequest("sendRichMessage", params); err != nil {
		log.Printf("rich: %v", err)
		sendHTML(bot, chatID, replyTo, card)
	}
}

func botUsername(bot *tgbotapi.BotAPI) string { return bot.Self.UserName }

func handleStart(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	if m.CommandArguments() == "help" {
		handleHelp(bot, m)
		return
	}
	lang := i18n.LangOf(m.From.LanguageCode)
	sendWithMarkup(bot, m.Chat.ID, startText(bot, m.From, lang), menuKeyboard(lang))
}

func handleHelp(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	lang := i18n.LangOf(m.From.LanguageCode)
	sendWithMarkup(bot, m.Chat.ID, i18n.Tr(lang, "help", "bot", botUsername(bot)), menuKeyboard(lang))
}

func handleIP(bot *tgbotapi.BotAPI, m *tgbotapi.Message) {
	lang := i18n.LangOf(m.From.LanguageCode)
	args := strings.TrimSpace(m.CommandArguments())
	if args == "" {
		sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "need_address"))
		return
	}
	replyLookup(bot, m, lang, args)
}

func replyLookup(bot *tgbotapi.BotAPI, m *tgbotapi.Message, lang, args string) {
	query := ipinfo.NormalizeQuery(args)
	if query == "" {
		sendHTML(bot, m.Chat.ID, m.MessageID, i18n.Tr(lang, "invalid_input"))
		return
	}
	info, whois, err := ipinfo.Lookup(query, lang, 0)
	if err != nil {
		sendHTML(bot, m.Chat.ID, m.MessageID, "❌ "+err.(*ipinfo.LookupErr).Text(lang))
		return
	}
	card := ipinfo.Format(info, query, lang, whois)
	if info.HasCoords() {
		title, address := ipinfo.VenueTexts(info)
		venue := tgbotapi.NewVenue(m.Chat.ID, title, address, *info.Lat, *info.Lon)
		venue.ReplyToMessageID = m.MessageID
		if _, err := bot.Send(venue); err != nil {
			log.Printf("venue for %s: %v", query, err)
		} else {
			sendRichCard(bot, m.Chat.ID, 0, lang, card, info)
			return
		}
	}
	sendRichCard(bot, m.Chat.ID, m.MessageID, lang, card, info)
}

var allowedUsers = map[int64]bool{
	6196125778: true,
}

func Start(botToken string) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка подключения бота: %v", err)
	}

	bot.Debug = false
	botAPI.Store(bot)
	token.Store(botToken)
	log.Printf("Бот авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	setupCommands(bot)

	for update := range updates {
		if u := update.SentFrom(); u == nil || !allowedUsers[u.ID] {
			continue
		}
		if update.InlineQuery != nil {
			go handleInline(bot, update.InlineQuery)
			continue
		}
		if update.CallbackQuery != nil {
			go handleCallback(bot, update.CallbackQuery)
			continue
		}
		m := update.Message
		if m == nil || m.Text == "" || m.From == nil {
			continue
		}
		if m.IsCommand() {
			switch m.Command() {
			case "ip", "info":
				go handleIP(bot, m)
			case "start":
				go handleStart(bot, m)
			case "help":
				go handleHelp(bot, m)
			}
			continue
		}

		go handleText(bot, m)
	}
}

var (
	latestInlineMu sync.Mutex
	latestInline   = map[int64]string{}
)

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
	lang := i18n.LangOf(q.From.LanguageCode)
	var results []any
	if query := ipinfo.NormalizeQuery(q.Query); query != "" {
		if !typingPaused(q) {
			return
		}
		if info, whois, err := ipinfo.Lookup(query, lang, ipinfo.APIReserve); err == nil {
			var parts []string
			for _, p := range []string{ipinfo.FormatPlace(info, false), info.Isp} {
				if p != "" {
					parts = append(parts, p)
				}
			}
			results = append(results, map[string]any{
				"type": "article", "id": "info",
				"title":       ipinfo.Flag(info.CountryCode) + " " + info.Query,
				"description": strings.Join(parts, " · "),
				"input_message_content": tgbotapi.InputTextMessageContent{
					Text: ipinfo.Format(info, query, lang, whois), ParseMode: tgbotapi.ModeHTML, DisableWebPagePreview: true,
				},
			})
			if info.HasCoords() {
				title, address := ipinfo.VenueTexts(info)
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
		params.AddInterface("button", map[string]string{"text": i18n.Tr(lang, "inline_hint"), "start_parameter": "help"})
	}
	if _, err := bot.MakeRequest("answerInlineQuery", params); err != nil {
		log.Printf("answerInlineQuery: %v", err)
	}
}

func setupCommands(bot *tgbotapi.BotAPI) {
	for _, lang := range i18n.Languages {
		cmds := []tgbotapi.BotCommand{
			{Command: "ip", Description: i18n.Tr(lang, "cmd_ip")},
			{Command: "help", Description: i18n.Tr(lang, "cmd_help")},
			{Command: "start", Description: i18n.Tr(lang, "cmd_start")},
		}
		code := lang
		if lang == i18n.DefaultLang {
			code = ""
		}
		cfg := tgbotapi.SetMyCommandsConfig{Commands: cmds, LanguageCode: code}
		if _, err := bot.Request(cfg); err != nil {
			log.Printf("setMyCommands %s: %v", lang, err)
		}
	}
}
