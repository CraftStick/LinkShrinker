package game

import (
	_ "embed"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:embed dino.html
var dino string

const (
	width  = 300
	height = 150
)

func Send(bot *tgbotapi.BotAPI, chatID int64, markup string) error {
	_, err := bot.MakeRequest("sendWidget", tgbotapi.Params{
		"chat_id":      strconv.FormatInt(chatID, 10),
		"html":         dino,
		"reply_markup": markup,
		"width":        strconv.Itoa(width),
		"height":       strconv.Itoa(height),
	})
	return err
}
