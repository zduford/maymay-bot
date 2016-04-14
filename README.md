# maymay-bot
maymay-bot is an implementation of the [Discord API](https://discordapp.com/developers/docs/intro). maymay-bot utilizes the [discordgo](https://github.com/bwmarrin/discordgo) library, a free and open source library. maymay-bot requires Go 1.4 or higher.

## Usage
maymay-bot is used to spam terrible sound bytes cause i'm garbo.

### Running the Bot
First First get the bot: `go get github.com/zduford/maymay-bot`
First install the bot: `go install github.com/zduford/maymay-bot`, then run the following command:

```
./bot -r "localhost:6379" -t "MY_BOT_ACCOUNT_TOKEN" -o OWNER_ID
```

To build of course get yerslef into the cmd/bot/ dir and do that whole `go install` jazz.


Note, I've still yet to remove all the redis references, so you'll still need a redis-server up to use this (for now, sorry ._.).

## Thanks
Thanks to hammerandchisel for making the airhornbot [hammerandchisel](https://github.com/hammerandchisel). <3
