package bot

import (
	"errors"
	"strconv"

	"tg-group-control-bot/internal/config"

	"tg-group-control-bot/internal/names"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

// UserCheck checks the presence of the user in DB and adds it in DB
func (b *Bot) UserCheck(user *tg.User) (config.User, error) {
	if user.IsBot {
		return config.User{}, errors.New("It is bot. ID: " + strconv.Itoa(user.ID))
	}

	// Get memoized value
	if mu, err := b.Memo.Get(user.ID); err == nil {
		// Try cast type
		if mcu, ok := mu.(config.User); ok {
			// Return memoized user if ok
			return mcu, nil
		}
	}

	u := config.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UserName:  user.UserName,
		Language:  user.LanguageCode,
		Bot:       user.IsBot,
	}

	new, cu, err := b.DB.UserCheck(u)
	if err != nil {
		return u, err
	}

	if new {
		b.Log.Info("Created new user ID: ", user.ID, " Name: ", names.FullUserName(user))
	}
	// Memoize confirmed user
	b.Memo.Set(u.ID, cu)

	return cu, nil
}
