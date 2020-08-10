package main

import (
	"errors"
	"strconv"

	"github.com/sattellite/tg-group-control-bot/internal/config"
	"github.com/sattellite/tg-group-control-bot/utils"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// checkUser checks the presence of the user in DB and adds it in DB
func checkUser(req Req, user *tg.User) (config.User, error) {
	log := req.App.Log.WithFields(logrus.Fields{
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

	new, ctxUser, err := req.App.DB.CheckUser(u)
	if err != nil {
		return u, err
	}

	if new {
		log.Info("Created new user ID: ", user.ID, " Name: ", utils.FullUserName(user))
	}

	return ctxUser, nil
}
