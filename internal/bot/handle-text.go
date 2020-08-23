package bot

import (
	"fmt"
	"strings"

	"tg-group-control-bot/internal/config"

	"tg-group-control-bot/internal/names"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

// HandleText start handling text messages
func (b *Bot) HandleText(message *tg.Message) error {
	// Cancel execution if command from bot or user is banned
	_, err := b.CheckUser(message.From)
	if err != nil {
		return err
	}

	switch {
	case message.NewChatMembers != nil:
		return b.userAddedHandler(message)
	case message.LeftChatMember != nil:
		return b.userLeftHandler(message)
	default:
		return b.textHandler(message)
	}
}

func (b *Bot) textHandler(message *tg.Message) error {
	// Message to chat with bot
	b.Log.Debug(message.From.ID, message.Chat.ID)
	if int64(message.From.ID) == message.Chat.ID {
		b.Log.Infof("Received message in bot chat from user %s with text `%s`", names.ShortUserName(message.From), message.Text)
		return b.checkAnswer(message)
	}
	b.Log.Infof("Received message in chat from user %s with text `%s`", names.ShortUserName(message.From), message.Text)
	// TODO Increment counter of user messages in chat
	return nil
}

func (b *Bot) checkAnswer(message *tg.Message) error {
	_, user, err := b.DB.CheckUser(config.User{ID: message.From.ID})
	if err != nil {
		return errors.Wrapf(err, "Failed get user info in checkAnswer for %s", names.ShortUserName(message.From))
	}

	if len(user.Chats) == 0 {
		return fmt.Errorf("No unconfirmed chats for %s", names.ShortUserName(message.From))
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
			// TODO Send error message to admins
			return fmt.Errorf("Failed restore new user privileges with code %d and error %s", resp.ErrorCode, resp.Description)
		}
		ref, err := b.DB.ConfirmChatUser(chatID, message.From.ID)
		if err != nil {
			return errors.Wrapf(err, "Error update user %d in storage for chat %s.", message.From.ID, names.ChatName(message.Chat))
		}
		// Delete confirmation message from group chat
		if ref.ChatID != 0 {
			_, err := b.API.DeleteMessage(tg.DeleteMessageConfig{
				ChatID:    ref.ChatID,
				MessageID: ref.MsgID,
			})
			if err != nil {
				b.Log.Errorf("Error delete confirmation message from chat %s %v", names.ChatName(message.Chat), err.Error())
				// TODO Send error message to admins
			}
		}
		// Delete chat from user's unconfirmed chats
		err = b.DB.DeleteUnconfirmedChat(chatID, message.From.ID)
		if err != nil {
			return errors.Wrapf(err, "Error delete user's(%d %s) unconfirmed chat %s %v", message.From.ID, names.ShortUserName(message.From), names.ChatName(message.Chat))
		}
		// Send success message to user in bot chat
		msg := b.TGMessageSuccess(chatID, message.Chat.ID)
		_, err = b.API.Send(msg)
		if err != nil {
			return errors.Wrapf(err, "Error sending success message in checkAnswer to user %s.", names.ShortUserName(message.From))
		}

		return nil
	}

	msg := b.TGMessageInvalid(chatID, message.Chat.ID)
	_, err = b.API.Send(msg)
	if err != nil {
		return errors.Wrapf(err, "Error sending invalid message in checkAnswer to user %s.", names.ShortUserName(message.From))
	}
	return nil
}

func (b *Bot) userAddedHandler(message *tg.Message) error {
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
		// message.From.ID and u.ID must be equal, it means that user was added by itself
		if message.From.ID != u.ID {
			continue
		}

		// If user was add by itself, than slice *message.NewChatMembers contains
		// only one user and then all "continue" can be replaced with "return error"

		isConfirmed, err := b.DB.UserConfirmed(message.Chat.ID, u.ID)
		if err != nil && err.Error() != mongo.ErrNoDocuments.Error() {
			// continue
			return errors.Wrap(err, "Failed check user confirmation in userAddedHandler.")
		}
		if !isConfirmed {
			err = b.DB.AddChatUser(message.Chat.ID, config.ChatUser{
				ID:        u.ID,
				Confirmed: !isNeedMessage,
				MsgCount:  0,
			})
			if err != nil {
				// continue
				return errors.Wrap(err, "Failed add user to chat.")
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
					err1 := errors.Wrapf(err, "Failed restrict new user privileges with code %d and error %s", resp.ErrorCode, resp.Description)
					b.Log.Error(err1)

					// Send message to admins that bot needs to be granted admin privileges
					ch, err := b.DB.GetChatInfo(message.Chat.ID)
					if err != nil {
						// continue
						return errors.Wrapf(err, "Error getting chat information %s.", message.Chat.ID)
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
							return errors.Wrapf(err, "Error sending message to admin %d in chat %s.", adm, chatTitle)
						}
					}
					return err1
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
					// continue
					return errors.Wrapf(err, "Error sending message to user %s.", names.FullUserName(message.From))
				}
				err = b.DB.UpdateConfirmReference(res.Chat.ID, res.MessageID, u.ID)
				if err != nil {
					// continue
					return errors.Wrapf(err, "Error update reference to confirm message for user %s.", names.FullUserName(message.From))
				}
			}
			// Add this chat to user's chats
			err = b.DB.AddUnconfirmedChat(message.Chat.ID, u.ID)
			if err != nil {
				return errors.Wrapf(err, "Failed add chat `%s` to user's %s unconfirmed chats.", names.ChatName(message.Chat), names.ShortUserName(message.From))
			}
		}
		return nil
	}
	return nil
}

func (b *Bot) userLeftHandler(message *tg.Message) error {
	// Remove from users list if user was not confirmed
	ref, err := b.DB.RemoveUnconfirmedChatUser(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		return errors.Wrapf(err, "Error remove unconfirmed user from chat %s", names.ChatName(message.Chat))
	}
	if ref.ChatID != 0 {
		// Remove message from chat
		_, err := b.API.DeleteMessage(tg.DeleteMessageConfig{
			ChatID:    ref.ChatID,
			MessageID: ref.MsgID,
		})
		if err != nil {
			b.Log.Errorf("Error delete confirmation message from chat %s %v", names.ChatName(message.Chat), err.Error())
		}
	}
	// Remove from admins list
	err = b.DB.RemoveChatAdmin(message.Chat.ID, message.LeftChatMember.ID)
	if err != nil {
		return errors.Wrapf(err, "Error delete user %d from admins from chat %s", message.LeftChatMember.ID, names.ChatName(message.Chat))
	}
	return nil
}
