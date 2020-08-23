package main

import "tg-group-control-bot/internal/bot"

func main() {
	app := bot.Init()
	app.Start()
}
