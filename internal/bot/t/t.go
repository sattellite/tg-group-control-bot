package t

import (
	"errors"
	"strconv"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/sattellite/tg-group-control-bot/internal/config"
	"github.com/sattellite/tg-group-control-bot/internal/names"
	"github.com/sattellite/tg-group-control-bot/internal/storage"
	"github.com/sirupsen/logrus"
)

// Bot is main type combining all needed data
type Bot struct {
	Config config.Config
	DB     *storage.Storage
	API    *tg.BotAPI
	Log    *logrus.Logger
}

// Req contains some data of request
type Req struct {
	ID   int64
	Time time.Time
	Bot  *Bot
}

// CheckUser checks the presence of the user in DB and adds it in DB
func (b *Bot) CheckUser(req Req, user *tg.User) (config.User, error) {
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      user,
	})

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
		log.Info("Created new user ID: ", user.ID, " Name: ", names.FullUserName(user))
	}

	return ctxUser, nil
}
