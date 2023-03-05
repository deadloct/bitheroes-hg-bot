package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/discord-squid-game/data"
	"github.com/deadloct/discord-squid-game/game"
	log "github.com/sirupsen/logrus"
)

const (
	CMD_PREFIX = "!"
	MODE_SQUID = "squid-game"
)

func init() {
	log.Info("Verbose logs enabled")
	log.SetLevel(log.DebugLevel)
}

func messageHandler(modes []*data.Mode) func(session *discordgo.Session, mc *discordgo.MessageCreate) {
	modeMap := make(map[string]*data.Mode)
	for _, mode := range modes {
		modeMap[mode.ID] = mode
	}

	return func(session *discordgo.Session, mc *discordgo.MessageCreate) {
		// Ignore messages from the bot
		if mc.Author.ID == session.State.User.ID {
			return
		}

		log.Debugf("received message: %#v", mc.Message)

		switch {
		case strings.HasPrefix(mc.Content, CMD_PREFIX+MODE_SQUID):
			game.NewGame(modeMap[MODE_SQUID]).Start(session, mc)
		default:
			session.ChannelMessageSend(mc.ChannelID, "Unknown command")
		}

		// ignore everything else
	}
}

func main() {
	session, err := discordgo.New("Bot " + os.Getenv("DISCORD_SQUID_GAME_AUTH_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	modes, err := data.LoadJSONData()
	if err != nil {
		log.Panic(err)
	}

	// Listen for server messages only
	session.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildMessageReactions | discordgo.IntentMessageContent
	session.AddHandler(messageHandler(modes))
	if err := session.Open(); err != nil {
		log.Panic(err)
	}

	log.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Info("Bot exiting...")
}
