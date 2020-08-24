package bot

import (
	"errors"
	"os"
	"time"

	"tg-group-control-bot/internal/config"
	"tg-group-control-bot/internal/memo"
	"tg-group-control-bot/internal/storage"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// Bot is main type combining all needed data
type Bot struct {
	Config config.Config
	DB     *storage.Storage
	API    *tg.BotAPI
	Log    *logrus.Logger
	Memo   *memo.Memo
}

// BotRequest contains some data of request
type BotRequest struct {
	ID   int64
	Time time.Time
}

type key string

const botReqKey key = "botreq"

// Init starts all services for bot
func Init() *Bot {
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

	memo := memo.New()

	return &Bot{
		Config: cfg,
		DB:     db,
		API:    bot,
		Log:    log,
		Memo:   memo,
	}
}

// Start starts polling for messages for bot
func (b *Bot) Start() {
	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.API.GetUpdatesChan(u)

	if err != nil {
		b.Log.Errorf("Failed get updates from Telegram. %v", err)
		os.Exit(1)
	}

	for update := range updates {
		switch {
		case update.EditedMessage != nil:
			go b.logger(update, b.Stub)
		case update.InlineQuery != nil:
			go b.logger(update, b.Stub)
		case update.ChosenInlineResult != nil:
			go b.logger(update, b.Stub)
		case update.CallbackQuery != nil:
			go b.logger(update, b.Stub)
		case update.Message.IsCommand():
			go b.logger(update, b.HandleCommand)
		default:
			go b.logger(update, b.HandleText)
		}
	}
}

func (b *Bot) logger(u tg.Update, h func(*tg.Message) error) {
	// Prepared data for logger
	t := time.Now()
	m, err := b.getMessage(&u)
	log := b.Log.WithFields(logrus.Fields{
		"requestID": t.UnixNano() / 1000,
		"user":      m.From,
	})
	mt := b.messageType(&u)
	if err != nil {
		log.Errorf("Error handling '%s' request %+v", mt, err)
		return
	}

	// Log before handling
	log.Infof("Started handling '%s' request", mt)
	// Log after handling
	defer log.Infof("Finished handling '%s' request. Duration %s", mt, time.Since(t))
	err = h(m)
	if err != nil {
		log.Errorf("Error handling '%s' request %+v", mt, err)
	}
}

func (b *Bot) getMessage(u *tg.Update) (*tg.Message, error) {
	switch {
	case u.EditedMessage != nil:
		return u.EditedMessage, nil
	case u.InlineQuery != nil:
		return &tg.Message{From: u.InlineQuery.From}, errors.New("tg.InlineQuery is not tg.Message")
	case u.ChosenInlineResult != nil:
		return &tg.Message{From: u.ChosenInlineResult.From}, errors.New("tg.ChosenInlineResult is not tg.Message")
	case u.CallbackQuery != nil:
		return &tg.Message{From: u.CallbackQuery.From}, errors.New("tg.CallbackQuery is not tg.Message")
	case u.Message != nil:
		return u.Message, nil
	default:
		return nil, errors.New("Unknown request type")
	}
}

func (b *Bot) messageType(u *tg.Update) string {
	switch {
	case u.EditedMessage != nil:
		return "edited message"
	case u.InlineQuery != nil:
		return "inline query"
	case u.ChosenInlineResult != nil:
		return "inline choice"
	case u.CallbackQuery != nil:
		return "callback"
	case u.Message.IsCommand():
		return "command"
	case u.Message.NewChatMembers != nil:
		return "new chat members"
	case u.Message.LeftChatMember != nil:
		return "left chat member"
	default:
		return "text message"
	}
}
