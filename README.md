# Chat control bot for telegram

Bot helps protect your telegram group chat from simple bots. Bot prohibits a new user
from write messages and read history until the user confirms that they are not a
bot.

It sends a message to group chat to confirm that the user is not a bot. The user
will be removed from group chat if they does not confirm that they is not a bot
within 24 hours.

A confirmation message will also be deleted from group chat if it was
successfully confirmed or after user was deleted by inactivity.

## Installation

Get a [bot token](https://core.telegram.org/bots) by chatting with
[BotFather](https://core.telegram.org/bots#6-botfather).

And you need to have MongoDB to store chat and some users data.

### Manual

1. `git clone https://github.com/sattellite/github.com/sattellite/tg-group-control-bot.git`
2. `cd tg-group-control-bot`
3. `go get -v -d ./...`
4. `go build -o group-control-bot cmd/grcbot/main.go`
5. `BOT_TOKEN=xxx MONGO_URL="mongodb://<user>:<password>@<host>:<port>/chat_control" ./group-control-bot`

> If you will copy binary file to other location then you need copy `locales` directory too.

### Docker

1. `git clone https://github.com/sattellite/github.com/sattellite/tg-group-control-bot.git`
2. `cd tg-group-control-bot`
3. Create `.env` file in root of project
4. `docker-compose up -d --build`

#### `.env` file for docker-compose

```
cat > .env <<EOC
BOT_TOKEN=123456:abcdef123456
ROOT_USER=mongo_admin_user
ROOT_PASS=mongo_admin_password
BOT_USER=mongo_db_user
BOT_PASS=mongo_db_password
EOC
```
