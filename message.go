package main

import (
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
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

func questionMessage(ctx Ctx, fromChatID, toChatID int64) *tg.MessageConfig {
	text := prepareText(ctx.App.DB.GetChatTitle(fromChatID), false)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

func invalidMessage(ctx Ctx, fromChatID, toChatID int64) *tg.MessageConfig {
	text := prepareText(ctx.App.DB.GetChatTitle(fromChatID), true)
	msg := tg.NewMessage(toChatID, text)

	return &msg
}

func successMessage(ctx Ctx, fromChatID, toChatID int64) *tg.MessageConfig {
	text := fmt.Sprintf("Вы прошли тест!\nВы получили доступ к чату %s", ctx.App.DB.GetChatTitle(fromChatID))
	msg := tg.NewMessage(toChatID, text)

	return &msg
}
