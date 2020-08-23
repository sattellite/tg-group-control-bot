package bot

import (
	"strconv"

	"tg-group-control-bot/internal/names"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

// HandleCommand start handling command message
func (b *Bot) HandleCommand(message *tg.Message) error {
	switch message.Command() {
	case "start":
		return b.askQuestion(message)
	default:
		return b.defaultCommand(message)
	}
}

func (b *Bot) askQuestion(message *tg.Message) error {
	chatID, err := strconv.ParseInt(message.CommandArguments(), 10, 64)
	if err != nil {
		return errors.Wrap(err, "Error parse chatID in askQuestion")
	}
	msg := b.TGMessageQuestion(chatID, message.Chat.ID)
	_, err = b.API.Send(msg)
	if err != nil {
		return errors.Wrapf(err, "Error sending message in askQuestion to user %s.", names.ShortUserName(message.From))
	}
	return nil
}

func (b *Bot) defaultCommand(message *tg.Message) error {
	b.Log.Warnf("Message from %s with unknown command %s with arguments %s", names.ShortUserName(message.From), message.Command(), message.CommandArguments())

	_, err := b.API.Send(tg.NewMessage(message.Chat.ID, "Неизвестная команда"))
	if err != nil {
		return errors.Wrapf(err, "Error sending message in defaultCommand to %d.", message.Chat.ID)
	}
	return nil
}
