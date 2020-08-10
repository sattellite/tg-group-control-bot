package names

import (
	"fmt"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sattellite/tg-group-control-bot/internal/config"
)

// FullUserName returns full name and nickname
func FullUserName(user *tg.User) string {
	if user.UserName != "" {
		return fmt.Sprintf("%s %s (%s)", user.FirstName, user.LastName, user.UserName)
	}
	return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
}

// ShortUserName returns or nickname or name of telegram user
func ShortUserName(user *tg.User) string {
	if user.UserName != "" {
		return fmt.Sprintf("@%s", user.UserName)
	}
	str := []string{user.FirstName, user.LastName}
	return strings.Join(str, " ")
}

// LocalUserShortName returns or nickname or name of local user
func LocalUserShortName(user config.User) string {
	if user.UserName != "" {
		return fmt.Sprintf("@%s", user.UserName)
	}
	str := []string{user.FirstName, user.LastName}
	return strings.Join(str, " ")
}

// ChatName returns chat name
func ChatName(ch *tg.Chat) string {
	chatTitle := ch.Title
	if ch.Type == "supergroup" && ch.UserName != "" {
		chatTitle = "@" + ch.UserName
	}
	return chatTitle
}
