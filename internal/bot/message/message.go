package message

import (
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/internal/bot/t"
)

func prepareText(chat string, isInvalid bool) string {
	invalid := "Неверный ответ. Попробуйте снова.\n\n"
	description := "Для получения доступа к чату " + chat + " ответьте на вопрос.\n\n"
	question := "Вы бот?"
	text := make([]string, 0)

	if isInvalid {
		text = append(text, invalid)
	}
	text = append(text, description, question)
	return strings.Join(text, "")
}

func Question(req t.Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := prepareText(req.Bot.DB.GetChatTitle(fromChatID), false)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

func Invalid(req t.Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := prepareText(req.Bot.DB.GetChatTitle(fromChatID), true)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

func Success(req t.Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := fmt.Sprintf("Вы прошли тест!\nВы получили доступ к чату %s", req.Bot.DB.GetChatTitle(fromChatID))
	msg := tg.NewMessage(toChatID, text)

	return &msg
}
