package bot

import (
	"strconv"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/internal/names"
	"github.com/sirupsen/logrus"
)

// HandleCommand start handling command message
func (b *Bot) HandleCommand(req Req, message *tg.Message) {
	switch message.Command() {
	case "start":
		b.askQuestion(req, message)
	default:
		b.defaultCommand(req, message)
	}
}

func (b *Bot) askQuestion(req Req, message *tg.Message) {
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})

	chatID, err := strconv.ParseInt(message.CommandArguments(), 10, 64)
	if err != nil {
		log.Errorf("Error parse chatID in askQuestion %v", err.Error())
		return
	}
	msg := b.TGMessageQuestion(req, chatID, message.Chat.ID)
	_, err = b.API.Send(msg)
	if err != nil {
		log.Errorf("Error sending message in askQuestion to user %s. %v", names.ShortUserName(message.From), err)
	}
}

func (b *Bot) defaultCommand(req Req, message *tg.Message) {
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})

	log.Warnf("Message from %s with unknown command %s with arguments %s", names.ShortUserName(message.From), message.Command(), message.CommandArguments())

	_, err := b.API.Send(tg.NewMessage(message.Chat.ID, "Неизвестная команда"))
	if err != nil {
		log.Errorf("Error sending message in defaultCommand to %d. %v", message.Chat.ID, err)
	}
}
