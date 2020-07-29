package main

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/config"
	"github.com/sattellite/tg-group-control-bot/utils"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

func handler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})

	// Cancel execution if command from bot or user is banned
	_, err := checkUser(ctx, message.From)
	if err != nil {
		log.Error(err)
		return
	}

	switch {
	case message.NewChatMembers != nil:
		userAddedHandler(ctx, message)
	case message.LeftChatMember != nil:
		userLeftHandler(ctx, message)
	default:
		textHandler(ctx, message)
	}
}

func textHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Debugln("textHandler")
	// TODO Увеличить счетчик сообщений для пользователя
	// utils.Dump(message)
}

func userAddedHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Infof("Got array of added users to group chat. Length: %d", len(*message.NewChatMembers))
	log.Info("Start handling added users")

	// TODO Добавить проверку, что если пользователь зашёл сам, то только тогда запускать обработку

	for _, u := range *message.NewChatMembers {
		isNeedMessage := true
		if u.ID == ctx.App.Bot.Self.ID {
			// Bot added to chat
			ctx.App.DB.UpdateChat(config.Chat{
				ID:     message.Chat.ID,
				Title:  message.Chat.Title,
				Type:   message.Chat.Type,
				Admins: []int{message.From.ID},
				Users: []config.ChatUser{{
					ID:        message.From.ID,
					Confirmed: true,
				}},
			})
			isNeedMessage = false
			// TODO Получить список админов чата
			// TODO Внести админов чата в список пользователей и в список админов
		}

		isConfirmed, err := ctx.App.DB.UserConfirmed(message.Chat.ID, u.ID)
		if err != nil && err.Error() != mongo.ErrNoDocuments.Error() {
			log.Errorf("Failed check user confirmation. %v", err)
			continue
		}
		if !isConfirmed {
			err = ctx.App.DB.AddChatUser(message.Chat.ID, config.ChatUser{
				ID:        u.ID,
				Confirmed: !isNeedMessage,
				MsgCount:  0,
			})
			if err != nil {
				log.Errorf("Failed add user to chat. %v", err)
				continue
			}
			if isNeedMessage {
				var f bool = false
				// Ограничить права пользователя
				resp, err := ctx.App.Bot.RestrictChatMember(tg.RestrictChatMemberConfig{
					ChatMemberConfig: tg.ChatMemberConfig{
						ChatID: message.Chat.ID,
						UserID: u.ID,
					},
					CanSendMessages:       &f,
					CanSendMediaMessages:  &f,
					CanSendOtherMessages:  &f,
					CanAddWebPagePreviews: &f,
				})
				if err != nil {
					log.Errorf("Failed restrict new user privileges with code %d and error %s", resp.ErrorCode, resp.Description)
					// TODO Отправить сообщение админам о необходимости повышения прав
					continue
				}
				// Формирование сообщения с кнопкой для перехода к тесту
				messageText := fmt.Sprintf("Привет %s\nТы в режиме только для чтения. Для того, чтобы получить полные права в этом чате надо пройти тест.\nНажми кнопку под этим сообщением, чтобы пройти тест.", utils.ShortUserName(&u))
				msg := tg.NewMessage(message.Chat.ID, messageText)
				msg.ParseMode = "Markdown"
				msg.ReplyToMessageID = message.MessageID

				buttons := tg.InlineKeyboardMarkup{
					InlineKeyboard: [][]tg.InlineKeyboardButton{},
				}
				testButton := tg.NewInlineKeyboardButtonURL(
					"Пройти тест",
					fmt.Sprintf("tg://resolve?domain=%s&start=%d", ctx.App.Bot.Self.UserName, message.Chat.ID),
				)
				buttons.InlineKeyboard = append(buttons.InlineKeyboard, tg.NewInlineKeyboardRow(testButton))
				msg.ReplyMarkup = buttons

				// Отправить сообщение для подтверждения
				res, err := ctx.App.Bot.Send(msg)
				if err != nil {
					log.Errorf("Error sending message to user %s. %v", utils.FullUserName(message.From), err)
					continue
				}
				err = ctx.App.DB.UpdateConfirmReference(res.Chat.ID, res.MessageID, u.ID)
				if err != nil {
					log.Errorf("Error update reference to confirm message for user %s. %v", utils.FullUserName(message.From), err)
					continue
				}
			}
		}
		log.Infof("Added user `%s` to chat `%s`", utils.ShortUserName(message.From), message.Chat.Title)
	}
}

func userLeftHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Debugln("userLeftHandler")
	// TODO Обработка удаления пользователя
	// TODO Удалить из списка админов, если он там присутствовал
	// TODO Удалить из списка пользователей, если  он не был подтвержден
	// utils.Dump(message.LeftChatMember)
}
