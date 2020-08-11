package main

import "github.com/sattellite/tg-group-control-bot/internal/bot"

func main() {
	app := bot.Init()
	bot.Start(app)
}
