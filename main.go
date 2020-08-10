package main

import (
	"os"
	"time"

	"github.com/sattellite/tg-group-control-bot/internal/config"
	"github.com/sattellite/tg-group-control-bot/internal/storage"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// App is main application context with common modules and data
type App struct {
	Config config.Config
	DB     *storage.Storage
	Bot    *tg.BotAPI
	Log    *logrus.Logger
}

// Req contains some data of request
type Req struct {
	ID   int64
	Time time.Time
	App  App
}

func main() {
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FullTimestamp:   true,
	}
	// log.ReportCaller = true

	var cfg config.Config
	if err := config.Create(&cfg); err != nil {
		log.Error("Error reading configuration.")
		log.Error(err)
		os.Exit(1)
	}
	if cfg.Debug {
		log.SetLevel(logrus.DebugLevel)
		// log.SetReportCaller(true)
	}
	log.Debug("Started in debug mode.")
	log.Info("Readed application configuration.")

	db, err := storage.New(&cfg, "chat_control")
	if err != nil {
		log.Error("Failed connect to database.")
		log.Error(err)
		os.Exit(2)
	}
	log.Info("Successfully connected to database.")

	bot, err := tg.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Error("Failed connect to Telegram.")
		log.Error(err)
		os.Exit(3)
	}
	bot.Debug = cfg.TelegramDebug

	app := App{
		Config: cfg,
		DB:     db,
		Bot:    bot,
	}

	log.Infof("Authorized on telegram account @%s", bot.Self.UserName)

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		// Create context for request
		reqTime := time.Now()
		req := Req{
			ID:   reqTime.UnixNano() / 1000,
			Time: reqTime,
			App:  app,
		}

		switch {
		case update.Message.IsCommand():
			log.WithFields(logrus.Fields{
				"requestID": req.ID,
				"user":      update.Message.From,
			}).Infof("Command request %s %s", update.Message.Command(), update.Message.CommandArguments())
			go command(req, update.Message)
		default:
			log.WithFields(logrus.Fields{
				"requestID": req.ID,
				"user":      update.Message.From,
			}).Infof("Text message request")
			go handler(req, update.Message)
		}
	}
}

// TODO Add timer function for delete old unconfirmed users
// TODO Add timer function for increment users messages in chats

func echo(app App, message *tg.Message) {
	msg := tg.NewMessage(message.Chat.ID, message.Text)
	msg.ReplyToMessageID = message.MessageID

	app.Bot.Send(msg)
}
