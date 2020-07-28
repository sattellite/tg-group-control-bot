# Prepare environment for development

## Define variables

This variables will be used in some places. You may store this variables in
environment or use it as plain text on each command.

> NOTE. You need change passwords below!

```shell
export MONGO_ROOT_USER=mongoadm
export MONGO_ROOT_PASS=aBP36Y6ChNGe3E
export MONGO_USER=bot
export MONGO_PASS=TWKwE8ub96FHvk
export MONGO_URL="mongodb://${MONGO_USER}:${MONGO_PASS}@localhost:27017/chat_control"
export BOT_TOKEN="123456789:ZZHQAaw1i1sTus3pFGHKX03fzLVpds7Oe8m"
```

## Create dockerized mongo database

Run mongo container:

```shell
docker run -itd --name mongo -p 127.0.0.1:27017:27017 -e MONGO_INITDB_ROOT_USERNAME=${MONGO_ROOT_USER} -e MONGO_INITDB_ROOT_PASSWORD=${MONGO_ROOT_PASS} mongo
```

Connect to mongo container:

```shell
docker exec -it mongo mongo --username mongoadm --password
```

And then create database:

```mongo
> db.runCommand({create:'chat_control'})
{ "ok" : 1 }
```

Select created database and create user for it:

```mongo
> use chat_control
switched to db chat_control

> db.createUser({user:'bot',pwd:'TWKwE8ub96FHvk',roles:[{role:'readWrite', db:'chat_control'}]})
Successfully added user: {
	"user" : "bot",
	"roles" : [
		{
			"role" : "readWrite",
			"db" : "chat_control"
		}
	]
}
```

## Run bot in development mode

Bot to run required two environment variables: `BOT_TOKEN` and `MONGO_URL`.
If this variables are exported already then just run `CompileDaemon`:

```shell
CompileDaemon -command="./tg-group-control-bot"
```

If variables not exported then it must be added before `CompileDaemon`:

```shell
BOT_TOKEN="123456789:ZZHQAaw1i1sTus3pFGHKX03fzLVpds7Oe8m" MONGO_URL="mongodb://bot:TWKwE8ub96FHvk@localhost:27017/chat_control" CompileDaemon -command="./tg-group-control-bot"
```

Environment variable | Type | Description
---|---|---
BOT_TOKEN | string | This token is authenticate bot in telegram. Token can be received by chatting with [BotFather](https://core.telegram.org/bots#6-botfather). **Required**
MONGO_URL | string | URL for connect to MongoDB. Mongo DB needs to store chat and some users data. **Required**
DEBUG | bool | Enable debug prints. Default **false**
TG_DEBUG | bool | Enable debug prints for telegram communications. Default **false**
