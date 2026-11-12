package i18n

import (
	"fmt"
	"strings"
)

const (
	DefaultLang = "en"
	examples    = "<code>8.8.8.8</code>, <code>2606:4700::1111</code>, <code>google.com</code>"
	waveEmoji   = `<tg-emoji emoji-id="5222459157084411333">👋</tg-emoji>`
	chatEmoji   = `<tg-emoji emoji-id="5294450432532323035">💬</tg-emoji>`
)

var texts = map[string]map[string]string{
	"en": {
		"start": waveEmoji + " Hello, {name}!\n\nI'm a handy toolbox: I shorten links, download videos from social networks without watermarks, show your IP, look up an <b>IP address</b> or a <b>domain</b> (location, provider and more) and have a little game to kill time. Pick an action below or just send me a link, an IP or a domain.\n\nExamples: {examples}\n\n" +
			chatEmoji + " Works in any chat too: type <code>@{bot} 8.8.8.8</code>",
		"help": "📚 <b>How to use</b>\n\n" +
			"Send a link to get a short one. A video link (TikTok, Instagram, YouTube, X, VK and others) gets you the file without a watermark (up to 50 MB). The My IP button shows your own address. Commands:\n\n" +
			"/ip <code>&lt;address&gt;</code> - look up an IP or domain\n" +
			"/info <code>&lt;address&gt;</code> - same as /ip\n" +
			"/start - welcome message\n" +
			"/help - this message\n\n" +
			"💬 <b>Inline mode:</b> in any chat type <code>@{bot} google.com</code> and pick the result.\n\n" +
			"⚙️ <b>Lookup shows:</b>\n" +
			"• Location on the map\n" +
			"• Country with flag, region, city and ZIP code\n" +
			"• Timezone with local time\n" +
			"• Internet provider, organization and AS\n" +
			"• Hostname (PTR)\n" +
			"• Domain registrar and registration dates (WHOIS)\n" +
			"• Mobile network flag",
		"need_address":     "Send an address after the command, e.g. <code>/ip 8.8.8.8</code>\nExamples: {examples}",
		"invalid_input":    "❌ That doesn't look like an IP address or domain.\nExamples: {examples}",
		"section_location": "Location",
		"section_network":  "Network",
		"local_time":       "now {time}",
		"continent":        "Continent",
		"zip":              "ZIP code",
		"timezone":         "Timezone",
		"coords":           "Coordinates",
		"isp":              "ISP",
		"org":              "Organization",
		"as":               "AS",
		"reverse":          "Hostname",
		"section_whois":    "WHOIS",
		"domain":           "Domain",
		"registrar":        "Registrar",
		"registered":       "Registered",
		"expires":          "Expires",
		"expired":          "Expired",
		"ago":              "{duration} ago",
		"in":               "in {duration}",
		"today":            "today",
		"date_format":      "2006-01-02",
		"tag_mobile":       "📱 Mobile network",
		"err_private":      "This is a private network address, it has no public location.",
		"err_reserved":     "This is a reserved address, it has no public location.",
		"err_invalid":      "Could not resolve this IP address or domain.",
		"err_rate_limit":   "Too many requests right now. Please try again in a minute.",
		"err_unavailable":  "Geolocation service is unavailable. Please try again later.",
		"err_unknown":      "Lookup failed: {message}",
		"inline_hint":      "ℹ️ Type an IP or domain · tap for help",
		"cmd_ip":           "Look up an IP address or domain",
		"cmd_help":         "How to use the bot",
		"cmd_start":        "Welcome message",
		"btn_short":        "Shorten a link",
		"btn_ip":           "IP / domain",
		"btn_game":         "Game",
		"game_private":     "The game works in a private chat with the bot.",
		"btn_cancel":       "◀️ Back",
		"btn_qr":           "QR code",
		"btn_copy":         "Copy",
		"btn_map":          "Map",
		"btn_myip":         "My IP",
		"video_wait":       "⏳ Downloading the video…",
		"video_fail":       "❌ Could not download the video: it is private, too large (over 50 MB) or the site blocked the request.",
		"menu":             "What do you want to do? Pick an action or just send a link, an IP or a domain.",
		"ask_url":          "Send the link to shorten.",
		"ask_ip":           "Send an IP address or a domain.\nExamples: {examples}",
		"short_result":     "<tg-emoji emoji-id=\"5222255236332168190\">🔗</tg-emoji> Short link:\n{url}",
		"short_denied":     "⛔ Link shortening is not available to you. IP and domain lookup works for everyone.",
	},
	"ru": {
		"start": waveEmoji + " Привет, {name}!\n\nЯ набор полезных штук: сокращаю ссылки, скачиваю видео из соцсетей без водяных знаков, показываю твой IP, ищу <b>IP-адрес</b> или <b>домен</b> (где находится, кто провайдер и многое другое) и есть небольшая игра, чтобы скоротать время. Выбери действие ниже или просто отправь ссылку, IP или домен.\n\nПримеры: {examples}\n\n" +
			chatEmoji + " Работаю и в любом чате: напиши <code>@{bot} 8.8.8.8</code>",
		"help": "📚 <b>Как пользоваться</b>\n\n" +
			"Отправь ссылку, чтобы получить короткую. Ссылка на видео (TikTok, Instagram, YouTube, X, VK и другие) вернёт файл без водяного знака (до 50 МБ). Кнопка «Мой IP» покажет твой адрес. Команды:\n\n" +
			"/ip <code>&lt;адрес&gt;</code> - найти IP или домен\n" +
			"/info <code>&lt;адрес&gt;</code> - то же, что /ip\n" +
			"/start - приветствие\n" +
			"/help - эта справка\n\n" +
			"💬 <b>Инлайн-режим:</b> в любом чате напиши <code>@{bot} google.com</code> и выбери результат.\n\n" +
			"⚙️ <b>Поиск показывает:</b>\n" +
			"• Точку на карте\n" +
			"• Страну с флагом, регион, город и индекс\n" +
			"• Часовой пояс с местным временем\n" +
			"• Провайдера, организацию и AS\n" +
			"• Имя хоста (PTR)\n" +
			"• Регистратора и даты регистрации домена (WHOIS)\n" +
			"• Признак мобильной сети",
		"need_address":     "Укажи адрес после команды, например <code>/ip 8.8.8.8</code>\nПримеры: {examples}",
		"invalid_input":    "❌ Это не похоже на IP-адрес или домен.\nПримеры: {examples}",
		"section_location": "Местоположение",
		"section_network":  "Сеть",
		"local_time":       "сейчас {time}",
		"continent":        "Континент",
		"zip":              "Индекс",
		"timezone":         "Часовой пояс",
		"coords":           "Координаты",
		"isp":              "Провайдер",
		"org":              "Организация",
		"as":               "AS",
		"reverse":          "Хост",
		"section_whois":    "WHOIS",
		"domain":           "Домен",
		"registrar":        "Регистратор",
		"registered":       "Зарегистрирован",
		"expires":          "Истекает",
		"expired":          "Истёк",
		"ago":              "{duration} назад",
		"in":               "через {duration}",
		"today":            "сегодня",
		"date_format":      "02.01.2006",
		"tag_mobile":       "📱 Мобильная сеть",
		"err_private":      "Это адрес частной сети, у него нет публичного местоположения.",
		"err_reserved":     "Это зарезервированный адрес, у него нет публичного местоположения.",
		"err_invalid":      "Не удалось найти такой IP-адрес или домен.",
		"err_rate_limit":   "Слишком много запросов. Попробуй через минуту.",
		"err_unavailable":  "Сервис геолокации недоступен. Попробуй позже.",
		"err_unknown":      "Не удалось выполнить запрос: {message}",
		"inline_hint":      "ℹ️ Введи IP или домен · нажми для справки",
		"cmd_ip":           "Найти IP-адрес или домен",
		"cmd_help":         "Как пользоваться ботом",
		"cmd_start":        "Приветствие",
		"btn_short":        "Сократить ссылку",
		"btn_ip":           "IP / домен",
		"btn_game":         "Игра",
		"game_private":     "Игра работает в личном чате с ботом.",
		"btn_cancel":       "◀️ Назад",
		"btn_qr":           "QR-код",
		"btn_copy":         "Копировать",
		"btn_map":          "Карта",
		"btn_myip":         "Мой IP",
		"video_wait":       "⏳ Скачиваю видео…",
		"video_fail":       "❌ Не удалось скачать видео: оно закрыто, больше 50 МБ или сайт заблокировал запрос.",
		"menu":             "Что сделать? Выбери действие или просто отправь ссылку, IP или домен.",
		"ask_url":          "Отправь ссылку, которую нужно сократить.",
		"ask_ip":           "Отправь IP-адрес или домен.\nПримеры: {examples}",
		"short_result":     "<tg-emoji emoji-id=\"5222255236332168190\">🔗</tg-emoji> Короткая ссылка:\n{url}",
		"short_denied":     "⛔ Сокращение ссылок тебе недоступно. Поиск по IP и домену работает для всех.",
	},
}

var Languages = []string{"en", "ru"}

var durationUnits = map[string]map[string][]string{
	"en": {"year": {"year", "years"}, "month": {"month", "months"}, "day": {"day", "days"}},
	"ru": {"year": {"год", "года", "лет"}, "month": {"месяц", "месяца", "месяцев"}, "day": {"день", "дня", "дней"}},
}

func LangOf(code string) string {
	if strings.HasPrefix(strings.ToLower(code), "ru") {
		return "ru"
	}
	return DefaultLang
}

func Tr(lang, key string, params ...string) string {
	text := texts[lang][key]
	if text == "" {
		text = texts[DefaultLang][key]
	}
	pairs := []string{"{examples}", examples}
	for i := 0; i+1 < len(params); i += 2 {
		pairs = append(pairs, "{"+params[i]+"}", params[i+1])
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

func pluralRu(n int, forms []string) string {
	n %= 100
	if n >= 11 && n <= 19 {
		return forms[2]
	}
	switch n % 10 {
	case 1:
		return forms[0]
	case 2, 3, 4:
		return forms[1]
	}
	return forms[2]
}

func FormatDuration(lang string, days int) string {
	if days < 0 {
		days = -days
	}
	unit, size := "day", 1
	for _, u := range []struct {
		name string
		size int
	}{{"year", 365}, {"month", 30}} {
		if days >= u.size {
			unit, size = u.name, u.size
			break
		}
	}
	count := days / size
	forms := durationUnits[lang][unit]
	var word string
	switch {
	case lang == "ru":
		word = pluralRu(count, forms)
	case count == 1:
		word = forms[0]
	default:
		word = forms[1]
	}
	return fmt.Sprintf("%d %s", count, word)
}
