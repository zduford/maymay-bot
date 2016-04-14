# maymay-bot
maymay-bot is an implementation of the [Discord API](https://discordapp.com/developers/docs/intro). maymay-bot utilizes the [discordgo](https://github.com/bwmarrin/discordgo) library, a free and open source library. maymay-bot requires Go 1.4 or higher.

## Usage
maymay-bot is used to spam terrible sound bytes cause i'm garbo.

### Running the Bot
First *get* the bot: `go get github.com/zduford/maymay-bot`

As well as getting the bot, you'll also need to get the other resources used to make this bot:

`go get github.com/Sirupsen/logrus`
`go get github.com/bwmarrin/discordgo`
`go get github.com/layeh/gopus`

As well as installing ffmpeg on your machine.


Then *install* the bot: `go install github.com/zduford/maymay-bot/cmd/bot`, make sure your audio folder is in the same direrctory as your executable, then run the following command:

```
./bot -t "MY_BOT_ACCOUNT_TOKEN" -o OWNER_ID
```

## Thanks
Thanks to hammerandchisel for making the airhornbot [hammerandchisel](https://github.com/hammerandchisel). <3
