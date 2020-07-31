package main

import (
	"fmt"
	"strings"

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
	// Message to chat with bot
	log.Debug(message.From.ID, message.Chat.ID)
	if int64(message.From.ID) == message.Chat.ID {
		log.Infof("Received message in bot chat from user %s with text `%s`", utils.ShortUserName(message.From), message.Text)
		checkAnswer(ctx, message)
		return
	}
	log.Infof("Received message in chat from user %s with text `%s`", utils.ShortUserName(message.From), message.Text)
	// TODO Increment counter of user messages in chat
	// utils.Dump(message)
}

func checkAnswer(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})

	_, user, err := ctx.App.DB.CheckUser(config.User{ID: message.From.ID})
	if err != nil {
		log.Errorf("Failed get user info in checkAnswer for %s %v", utils.ShortUserName(message.From), err.Error())
	}

	if len(user.Chats) == 0 {
		log.Errorf("No unconfirmed chats for %s", utils.ShortUserName(message.From))
		return
	}

	chatID := user.Chats[len(user.Chats)-1]

	lowerCasedText := strings.ToLower(message.Text)
	if lowerCasedText == "нет" || lowerCasedText == "no" {
		var t bool = true
		// Grant user permissions
		resp, err := ctx.App.Bot.RestrictChatMember(tg.RestrictChatMemberConfig{
			ChatMemberConfig: tg.ChatMemberConfig{
				ChatID: chatID,
				UserID: message.From.ID,
			},
			CanSendMessages:       &t,
			CanSendMediaMessages:  &t,
			CanSendOtherMessages:  &t,
			CanAddWebPagePreviews: &t,
		})
		if err != nil {
			log.Errorf("Failed restore new user privileges with code %d and error %s", resp.ErrorCode, resp.Description)
			// TODO Send error message to admins
			return
		}
		ref, err := ctx.App.DB.ConfirmChatUser(chatID, message.From.ID)
		if err != nil {
			log.Errorf("Error delete user %d from admins from chat %s %v", message.LeftChatMember.ID, utils.ChatName(message.Chat), err.Error())
			return
		}
		// Delete confirmation message from group chat
		if ref.ChatID != 0 {
			_, err := ctx.App.Bot.DeleteMessage(tg.DeleteMessageConfig{
				ChatID:    ref.ChatID,
				MessageID: ref.MsgID,
			})
			if err != nil {
				log.Errorf("Error delete confirmation message from chat %s %v", utils.ChatName(message.Chat), err.Error())
			}
		}
		// Delete chat from user's unconfirmed chats
		err = ctx.App.DB.DeleteUnconfirmedChat(chatID, message.From.ID)
		if err != nil {
			log.Errorf("Error delete user's(%d) unconfirmed chat %s %v", message.From.ID, utils.ChatName(message.Chat), err.Error())
			return
		}
		// Send success message to user in bot chat
		msg := successMessage(ctx, chatID, message.Chat.ID)
		_, err = ctx.App.Bot.Send(msg)
		if err != nil {
			log.Errorf("Error sending success message in checkAnswer to user %s. %v", utils.ShortUserName(message.From), err)
		}

		return
	}

	msg := invalidMessage(ctx, chatID, message.Chat.ID)
	_, err = ctx.App.Bot.Send(msg)
	if err != nil {
		log.Errorf("Error sending invalid message in checkAnswer to user %s. %v", utils.ShortUserName(message.From), err)
	}
}

func userAddedHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Infof("Got array of added users to group chat. Length: %d", len(*message.NewChatMembers))
	log.Info("Start handling added users")

	for _, u := range *message.NewChatMembers {
		isNeedMessage := true
		if u.ID == ctx.App.Bot.Self.ID {
			// Bot added to chat
			ctx.App.DB.UpdateChat(config.Chat{
				ID:       message.Chat.ID,
				Title:    message.Chat.Title,
				UserName: message.Chat.UserName,
				Type:     message.Chat.Type,
				Admins:   []int{message.From.ID},
				Users: []config.ChatUser{{
					ID:        message.From.ID,
					Confirmed: true,
				}},
			})
			isNeedMessage = false
		}

		// Stop handling if someone from chat added users
		if message.From.ID != u.ID {
			continue
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
				// Restrict user permissions
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
					// Send message to admins that bot needs to be granted admin privileges
					ch, err := ctx.App.DB.GetChatInfo(message.Chat.ID)
					if err != nil {
						log.Errorf("Error getting chat information %s. %v", message.Chat.ID, err)
						continue
					}
					chatTitle := ch.Title
					if ch.Type == "supergroup" && ch.UserName != "" {
						chatTitle = "@" + ch.UserName
					}
					adminText := fmt.Sprintf("Grant admin privileges to bot @%s in chat %s", ctx.App.Bot.Self.UserName, chatTitle)
					for _, adm := range ch.Admins {
						msg := tg.NewMessage(int64(adm), adminText)
						_, err := ctx.App.Bot.Send(msg)
						if err != nil {
							log.Errorf("Error sending message to admin %d in chat %s. %v", adm, chatTitle, err)
						}
					}
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
			// Add this chat to user's chats
			err = ctx.App.DB.AddUnconfirmedChat(message.Chat.ID, u.ID)
		}
		log.Infof("Added user `%s` to chat `%s`", utils.ShortUserName(message.From), utils.ChatName(message.Chat))
	}
}

func userLeftHandler(ctx Ctx, message *tg.Message) {
	log := ctx.Log.WithFields(logrus.Fields{
		"requestID": ctx.RequestID,
		"user":      message.From,
	})
	log.Infof("Start handling left user %s from chat %s", utils.ShortUserName(message.From), utils.ChatName(message.Chat))
	// Remove from users list if user was not confirmed
	ref, err := ctx.App.DB.RemoveUnconfirmedChatUser(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		log.Errorf("Error remove unconfirmed user from chat %s %v", utils.ChatName(message.Chat), err.Error())
	}
	if ref.ChatID != 0 {
		// Remove message from chat
		_, err := ctx.App.Bot.DeleteMessage(tg.DeleteMessageConfig{
			ChatID:    ref.ChatID,
			MessageID: ref.MsgID,
		})
		if err != nil {
			log.Errorf("Error delete confirmation message from chat %s %v", utils.ChatName(message.Chat), err.Error())
		}
	}
	// Remove from admins list
	err = ctx.App.DB.RemoveChatAdmin(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		log.Errorf("Error delete user %d from admins from chat %s %v", message.LeftChatMember.ID, utils.ChatName(message.Chat), err.Error())
	}
}
