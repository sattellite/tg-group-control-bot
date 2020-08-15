package bot

import (
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (b *Bot) prepareText(chat string, isInvalid bool) string {
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

// TGMessageQuestion returns telegram message with question for confirm
func (b *Bot) TGMessageQuestion(req Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := b.prepareText(b.DB.GetChatTitle(fromChatID), false)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

// TGMessageInvalid returns telegram message with text about incorrect answer
func (b *Bot) TGMessageInvalid(req Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := b.prepareText(b.DB.GetChatTitle(fromChatID), true)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

// TGMessageSuccess returns telegram message with success text
func (b *Bot) TGMessageSuccess(req Req, fromChatID, toChatID int64) *tg.MessageConfig {
	text := fmt.Sprintf("Вы прошли тест!\nВы получили доступ к чату %s", b.DB.GetChatTitle(fromChatID))
	msg := tg.NewMessage(toChatID, text)

	return &msg
}
