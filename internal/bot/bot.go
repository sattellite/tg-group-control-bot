package bot

import (
	"os"
	"time"

	"github.com/sattellite/tg-group-control-bot/internal/bot/handlers/command"
	"github.com/sattellite/tg-group-control-bot/internal/bot/handlers/text"
	"github.com/sattellite/tg-group-control-bot/internal/bot/t"
	"github.com/sattellite/tg-group-control-bot/internal/config"
	"github.com/sattellite/tg-group-control-bot/internal/storage"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// Init starts all services for bot
func Init() *t.Bot {
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FullTimestamp:   true,
	}

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
		os.Exit(1)
	}
	log.Info("Successfully connected to database.")

	bot, err := tg.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Errorf("Failed connect to Telegram. %v", err)
		os.Exit(1)
	}
	log.Infof("Authorized on telegram account @%s", bot.Self.UserName)

	bot.Debug = cfg.TelegramDebug

	return &t.Bot{
		Config: cfg,
		DB:     db,
		API:    bot,
		Log:    log,
	}
}

// Start starts polling for messages for bot
func Start(bot *t.Bot) {
	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.API.GetUpdatesChan(u)

	if err != nil {
		bot.Log.Errorf("Failed get updates from Telegram. %v", err)
		os.Exit(1)
	}

	for update := range updates {
		// Create context for request
		reqTime := time.Now()
		req := t.Req{
			ID:   reqTime.UnixNano() / 1000,
			Time: reqTime,
			Bot:  bot,
		}

		switch {
		case update.Message.IsCommand():
			bot.Log.WithFields(logrus.Fields{
				"requestID": req.ID,
				"user":      update.Message.From,
			}).Infof("Command request %s %s", update.Message.Command(), update.Message.CommandArguments())
			go command.Handle(req, update.Message)
		default:
			bot.Log.WithFields(logrus.Fields{
				"requestID": req.ID,
				"user":      update.Message.From,
			}).Infof("Text message request")
			go text.Handle(req, update.Message)
		}
	}
}
