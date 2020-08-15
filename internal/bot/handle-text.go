package bot

import (
	"context"
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/internal/config"
	"github.com/sattellite/tg-group-control-bot/internal/names"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// HandleText start handling text messages
func (b *Bot) HandleText(ctx context.Context, message *tg.Message) {
	req, ok := b.reqFromContext(ctx)
	if !ok {
		b.Log.Errorf("Failed to get request values from context in HandleText")
		return
	}
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})

	// Cancel execution if command from bot or user is banned
	_, err := b.CheckUser(ctx, message.From)
	if err != nil {
		log.Error(err)
		return
	}

	switch {
	case message.NewChatMembers != nil:
		b.userAddedHandler(ctx, message)
	case message.LeftChatMember != nil:
		b.userLeftHandler(ctx, message)
	default:
		b.textHandler(ctx, message)
	}
}

func (b *Bot) textHandler(ctx context.Context, message *tg.Message) {
	req, ok := b.reqFromContext(ctx)
	if !ok {
		b.Log.Errorf("Failed to get request values from context in textHandler")
		return
	}
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})
	// Message to chat with bot
	log.Debug(message.From.ID, message.Chat.ID)
	if int64(message.From.ID) == message.Chat.ID {
		log.Infof("Received message in bot chat from user %s with text `%s`", names.ShortUserName(message.From), message.Text)
		b.checkAnswer(ctx, message)
		return
	}
	log.Infof("Received message in chat from user %s with text `%s`", names.ShortUserName(message.From), message.Text)
	// TODO Increment counter of user messages in chat
}

func (b *Bot) checkAnswer(ctx context.Context, message *tg.Message) {
	req, ok := b.reqFromContext(ctx)
	if !ok {
		b.Log.Errorf("Failed to get request values from context in checkAnswer")
		return
	}
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})

	_, user, err := b.DB.CheckUser(config.User{ID: message.From.ID})
	if err != nil {
		log.Errorf("Failed get user info in checkAnswer for %s %v", names.ShortUserName(message.From), err.Error())
	}

	if len(user.Chats) == 0 {
		log.Errorf("No unconfirmed chats for %s", names.ShortUserName(message.From))
		return
	}

	chatID := user.Chats[len(user.Chats)-1]

	lowerCasedText := strings.ToLower(message.Text)
	if lowerCasedText == "нет" || lowerCasedText == "no" {
		var t bool = true
		// Grant user permissions
		resp, err := b.API.RestrictChatMember(tg.RestrictChatMemberConfig{
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
		ref, err := b.DB.ConfirmChatUser(chatID, message.From.ID)
		if err != nil {
			log.Errorf("Error update user %d in storage for chat %s. %v", message.From.ID, names.ChatName(message.Chat), err.Error())
			return
		}
		// Delete confirmation message from group chat
		if ref.ChatID != 0 {
			_, err := b.API.DeleteMessage(tg.DeleteMessageConfig{
				ChatID:    ref.ChatID,
				MessageID: ref.MsgID,
			})
			if err != nil {
				log.Errorf("Error delete confirmation message from chat %s %v", names.ChatName(message.Chat), err.Error())
			}
		}
		// Delete chat from user's unconfirmed chats
		err = b.DB.DeleteUnconfirmedChat(chatID, message.From.ID)
		if err != nil {
			log.Errorf("Error delete user's(%d) unconfirmed chat %s %v", message.From.ID, names.ChatName(message.Chat), err.Error())
			return
		}
		// Send success message to user in bot chat
		msg := b.TGMessageSuccess(chatID, message.Chat.ID)
		_, err = b.API.Send(msg)
		if err != nil {
			log.Errorf("Error sending success message in checkAnswer to user %s. %v", names.ShortUserName(message.From), err)
		}

		return
	}

	msg := b.TGMessageInvalid(chatID, message.Chat.ID)
	_, err = b.API.Send(msg)
	if err != nil {
		log.Errorf("Error sending invalid message in checkAnswer to user %s. %v", names.ShortUserName(message.From), err)
	}
}

func (b *Bot) userAddedHandler(ctx context.Context, message *tg.Message) {
	req, ok := b.reqFromContext(ctx)
	if !ok {
		b.Log.Errorf("Failed to get request values from context in userAddedHandler")
		return
	}
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})
	log.Infof("Got array of added users to group chat. Length: %d", len(*message.NewChatMembers))
	log.Info("Start handling added users")

	for _, u := range *message.NewChatMembers {
		isNeedMessage := true
		if u.ID == b.API.Self.ID {
			// Bot added to chat
			b.DB.UpdateChat(config.Chat{
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

		isConfirmed, err := b.DB.UserConfirmed(message.Chat.ID, u.ID)
		if err != nil && err.Error() != mongo.ErrNoDocuments.Error() {
			log.Errorf("Failed check user confirmation. %v", err)
			continue
		}
		if !isConfirmed {
			err = b.DB.AddChatUser(message.Chat.ID, config.ChatUser{
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
				resp, err := b.API.RestrictChatMember(tg.RestrictChatMemberConfig{
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
					ch, err := b.DB.GetChatInfo(message.Chat.ID)
					if err != nil {
						log.Errorf("Error getting chat information %s. %v", message.Chat.ID, err)
						continue
					}
					chatTitle := ch.Title
					if ch.Type == "supergroup" && ch.UserName != "" {
						chatTitle = "@" + ch.UserName
					}
					adminText := fmt.Sprintf("Grant admin privileges to bot @%s in chat %s", b.API.Self.UserName, chatTitle)
					for _, adm := range ch.Admins {
						msg := tg.NewMessage(int64(adm), adminText)
						_, err := b.API.Send(msg)
						if err != nil {
							log.Errorf("Error sending message to admin %d in chat %s. %v", adm, chatTitle, err)
						}
					}
					continue
				}
				// Формирование сообщения с кнопкой для перехода к тесту
				messageText := fmt.Sprintf("Привет %s\nТы в режиме только для чтения. Для того, чтобы получить полные права в этом чате надо пройти тест.\nНажми кнопку под этим сообщением, чтобы пройти тест.", names.ShortUserName(&u))
				msg := tg.NewMessage(message.Chat.ID, messageText)
				msg.ParseMode = "Markdown"
				msg.ReplyToMessageID = message.MessageID

				buttons := tg.InlineKeyboardMarkup{
					InlineKeyboard: [][]tg.InlineKeyboardButton{},
				}
				testButton := tg.NewInlineKeyboardButtonURL(
					"Пройти тест",
					fmt.Sprintf("tg://resolve?domain=%s&start=%d", b.API.Self.UserName, message.Chat.ID),
				)
				buttons.InlineKeyboard = append(buttons.InlineKeyboard, tg.NewInlineKeyboardRow(testButton))
				msg.ReplyMarkup = buttons

				// Отправить сообщение для подтверждения
				res, err := b.API.Send(msg)
				if err != nil {
					log.Errorf("Error sending message to user %s. %v", names.FullUserName(message.From), err)
					continue
				}
				err = b.DB.UpdateConfirmReference(res.Chat.ID, res.MessageID, u.ID)
				if err != nil {
					log.Errorf("Error update reference to confirm message for user %s. %v", names.FullUserName(message.From), err)
					continue
				}
			}
			// Add this chat to user's chats
			err = b.DB.AddUnconfirmedChat(message.Chat.ID, u.ID)
		}
		log.Infof("Added user `%s` to chat `%s`", names.ShortUserName(message.From), names.ChatName(message.Chat))
	}
}

func (b *Bot) userLeftHandler(ctx context.Context, message *tg.Message) {
	req, ok := b.reqFromContext(ctx)
	if !ok {
		b.Log.Errorf("Failed to get request values from context in userLeftHandler")
		return
	}
	log := b.Log.WithFields(logrus.Fields{
		"requestID": req.ID,
		"user":      message.From,
	})
	log.Infof("Start handling left user %s from chat %s", names.ShortUserName(message.From), names.ChatName(message.Chat))
	// Remove from users list if user was not confirmed
	ref, err := b.DB.RemoveUnconfirmedChatUser(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		log.Errorf("Error remove unconfirmed user from chat %s %v", names.ChatName(message.Chat), err.Error())
	}
	if ref.ChatID != 0 {
		// Remove message from chat
		_, err := b.API.DeleteMessage(tg.DeleteMessageConfig{
			ChatID:    ref.ChatID,
			MessageID: ref.MsgID,
		})
		if err != nil {
			log.Errorf("Error delete confirmation message from chat %s %v", names.ChatName(message.Chat), err.Error())
		}
	}
	// Remove from admins list
	err = b.DB.RemoveChatAdmin(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		log.Errorf("Error delete user %d from admins from chat %s %v", message.LeftChatMember.ID, names.ChatName(message.Chat), err.Error())
	}
}
