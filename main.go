package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/cmd"
	"github.com/deadloct/bitheroes-hg-bot/game"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.Info("verbose logs enabled")
	log.SetLevel(log.DebugLevel)

	settings.ImportData()
}

func main() {
	session, err := discordgo.New("Bot " + os.Getenv("BITHEROES_HG_BOT_AUTH_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Listen for server messages only
	session.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentGuildMessageReactions | discordgo.IntentMessageContent
	session.AddHandler(cmd.CommandHandler)
	session.AddHandler(game.ManagerInstance(session).ReactionHandler)
	if err := session.Open(); err != nil {
		log.Panic(err)
	}

	err = cmd.RegisterCommands(session)
	if err != nil {
		log.Panicf("error registering slash commands: %v", err)
	}
	defer cmd.DeregisterCommmands(session)

	log.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Info("Bot exiting...")
}
