package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/utils"
	"github.com/sirupsen/logrus"
)

func handler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})

	// Cancel execution if command from bot or user is banned
	_, err := checkUser(ctx, message.From)
	if err != nil {
		log.Error(err)
		return
	}

	switch {
	case message.NewChatMembers != nil:
		userAddedHandler(ctx, message)
	case message.LeftChatMember != nil:
		userLeftHandler(ctx, message)
	default:
		textHandler(ctx, message)
	}
}

func textHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Debugln("textHandler")
	// TODO Увеличить счетчик сообщений для пользователя
	utils.Dump(message)
}

func userAddedHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Debugln("userAddedHandler")
	// TODO Обработка добавления пользователя
	utils.Dump(message.NewChatMembers)
}

func userLeftHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Debugln("userLeftHandler")
	// TODO Обработка удаления пользователя
	utils.Dump(message.LeftChatMember)
}
