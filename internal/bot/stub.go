package bot

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

// Stub handler for unsupported messages
func (b *Bot) Stub(message *tg.Message) error {
	return errors.New("Unsupported message")
}
