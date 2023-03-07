package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/game"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.Info("verbose logs enabled")
	log.SetLevel(log.DebugLevel)
}

func messageHandler(session *discordgo.Session, mc *discordgo.MessageCreate) {
	// Ignore messages from the bot
	if mc.Author.ID == session.State.User.ID {
		return
	}

	switch {
	case strings.HasPrefix(mc.Content, settings.CMDHG):
		if err := game.ManagerInstance(session).StartGame(mc.ChannelID, mc.Content, mc.Author); err != nil {
			log.Errorf("error starting game: %v", err)
		}
	default:
		session.ChannelMessageSend(mc.ChannelID, "Unknown command")
	}

	// ignore everything else
}

func main() {
	session, err := discordgo.New("Bot " + os.Getenv("BITHEROES_HG_BOT_AUTH_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Listen for server messages only
	session.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildMessageReactions | discordgo.IntentMessageContent
	session.AddHandler(messageHandler)
	if err := session.Open(); err != nil {
		log.Panic(err)
	}

	log.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Info("Bot exiting...")
}
