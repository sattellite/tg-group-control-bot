package main

import (
	"strconv"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/utils"
	"github.com/sirupsen/logrus"
)

func command(ctx Ctx, message *tg.Message) {
	switch message.Command() {
	case "start":
		askQuestion(ctx, message)
	default:
		defaultCommand(ctx, message)
	}
}

func askQuestion(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})

	chatID, err := strconv.ParseInt(message.CommandArguments(), 10, 64)
	if err != nil {
		log.Errorf("Error parse chatID in askQuestion %v", err.Error())
		return
	}
	msg := questionMessage(ctx, chatID, message.Chat.ID)
	_, err = ctx.App.Bot.Send(msg)
	if err != nil {
		log.Errorf("Error sending message in askQuestion to user %s. %v", utils.ShortUserName(message.From), err)
	}
}

func defaultCommand(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})

	log.Warnf("Message from %s with unknown command %s with arguments %s", utils.ShortUserName(message.From), message.Command(), message.CommandArguments())

	_, err := ctx.App.Bot.Send(tg.NewMessage(message.Chat.ID, "Неизвестная команда"))
	if err != nil {
		log.Errorf("Error sending message in defaultCommand to %d. %v", message.Chat.ID, err)
	}
}
