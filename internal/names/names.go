package names

import (
	"fmt"
	"strings"

	"tg-group-control-bot/internal/config"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

// FullUserName returns full name and nickname
func FullUserName(user *tg.User) string {
	if user.UserName != "" {
		return replaceUnderscore(fmt.Sprintf("%s %s (%s)", user.FirstName, user.LastName, user.UserName))
	}
	return replaceUnderscore(fmt.Sprintf("%s %s", user.FirstName, user.LastName))
}

// ShortUserName returns or nickname or name of telegram user
func ShortUserName(user *tg.User) string {
	if user.UserName != "" {
		return replaceUnderscore(fmt.Sprintf("@%s", user.UserName))
	}
	str := []string{user.FirstName, user.LastName}
	return replaceUnderscore(strings.Join(str, " "))
}

// LocalUserShortName returns or nickname or name of local user
func LocalUserShortName(user config.User) string {
	if user.UserName != "" {
		return replaceUnderscore(fmt.Sprintf("@%s", user.UserName))
	}
	str := []string{user.FirstName, user.LastName}
	return replaceUnderscore(strings.Join(str, " "))
}

// ChatName returns chat name
func ChatName(ch *tg.Chat) string {
	chatTitle := ch.Title
	if ch.Type == "supergroup" && ch.UserName != "" {
		chatTitle = "@" + ch.UserName
	}
	return replaceUnderscore(chatTitle)
}

func replaceUnderscore(str string) string {
	return strings.ReplaceAll(str, "_", "\\_")
}
