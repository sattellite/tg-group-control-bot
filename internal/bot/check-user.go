package bot

import (
	"errors"
	"strconv"

	"tg-group-control-bot/internal/config"

	"tg-group-control-bot/internal/names"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

// CheckUser checks the presence of the user in DB and adds it in DB
func (b *Bot) CheckUser(user *tg.User) (config.User, error) {
	if user.IsBot {
		return config.User{}, errors.New("It is bot. ID: " + strconv.Itoa(user.ID))
	}

	u := config.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UserName:  user.UserName,
		Language:  user.LanguageCode,
		Bot:       user.IsBot,
	}

	new, ctxUser, err := b.DB.CheckUser(u)
	if err != nil {
		return u, err
	}

	if new {
		b.Log.Info("Created new user ID: ", user.ID, " Name: ", names.FullUserName(user))
	}

	return ctxUser, nil
}
