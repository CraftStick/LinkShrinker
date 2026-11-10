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

func HTML() string { return dino }

func Send(bot *tgbotapi.BotAPI, chatID int64, caption string) error {
	_, err := bot.MakeRequest("sendWidget", tgbotapi.Params{
		"chat_id": strconv.FormatInt(chatID, 10),
		"html":    dino,
		"caption": caption,
		"width":   strconv.Itoa(width),
		"height":  strconv.Itoa(height),
	})
	return err
}
