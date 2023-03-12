package cmd

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/game"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

const (
	CommandPrefix          = "hg-"
	CommandHelp            = CommandPrefix + "help"
	CommandStart           = CommandPrefix + "start"
	CommandStartOptionWait = "wait"
	CommandCancel          = CommandPrefix + "cancel"
	CommandClear           = CommandPrefix + "clear"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        CommandHelp,
		Description: "Explains how to use this bot",
	},
	{
		Name:        CommandStart,
		Description: "Starts a Hunger Games event",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        CommandStartOptionWait,
				Description: "Seconds to wait for reactions before starting game (Default: 60 seconds)",
				Required:    false,
			},
		},
	},
	{
		Name:        CommandCancel,
		Description: "Cancel the active Hunger Games event in this channel",
	},
	{
		Name:        CommandClear,
		Description: "Clear bot messages in this channel",
	},
}

func RegisterCommands(session *discordgo.Session) error {
	log.Info("registering commands")

	for _, v := range commands {
		if _, err := session.ApplicationCommandCreate(session.State.User.ID, "", v); err != nil {
			log.Errorf("error creating command %v: %v", v.Name, err)
			return err
		}

		log.Infof("registered command %v", v.Name)
	}

	log.Info("finished registering commands")

	return nil
}

func DeregisterCommmands(session *discordgo.Session) error {
	existingCommands, err := session.ApplicationCommands(session.State.User.ID, "")
	if err != nil {
		log.Errorf("could not retrieve existing commands: %v", err)
	}

	log.Info("deregistering commands")

	for _, v := range existingCommands {
		log.Infof("deregistering command %v", v.Name)
		if err := session.ApplicationCommandDelete(session.State.User.ID, "", v.ID); err != nil {
			log.Infof("failed to deregister command %v: %v", v.Name, err)
			continue
		}

		log.Infof("deregistered command %v", v.Name)
	}

	log.Info("finished deregistering commands")
	return nil
}

func CommandHandler(session *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.Member == nil {
		log.Infof("user attempted to run the bot from outside a channel: %v", ic.User.ID)
		content := "Citizens must sponsor a new Hunger Games from a channel."

		err := session.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content},
		})

		if err != nil {
			log.Errorf("error when user attempted to run commands outside a channel: %v", err)
		}

		return
	}

	session.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "Command acknowledged. Engaging the capitol of Panem."},
	})

	options := ic.ApplicationCommandData().Options

	v := ic.ApplicationCommandData().Name
	switch v {
	case CommandHelp:
		session.ChannelMessageSend(ic.ChannelID, "This command is not yet supported. Panem issues our sincerest apologies and politely requests your obedience.")

	case CommandStart:
		delay := settings.DefaultStartDelay

		for _, option := range options {
			switch option.Name {
			case CommandStartOptionWait:
				v := int(option.IntValue())
				switch {
				case v < settings.MinimumStartDelay:
					msg := fmt.Sprintf("The delay of %v is much too short. Hunger Games will wait for %v instead.", v, settings.DefaultStartDelay)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				case v > settings.MaximumStartDelay:
					msg := fmt.Sprintf("The delay of %v is much too long. Hunger Games will wait for %v instead.", v, settings.DefaultStartDelay)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				default:
					delay = time.Duration(v) * time.Second
				}
			}
		}

		if err := game.ManagerInstance(session).StartGame(ic.ChannelID, delay, ic.Member.User); err != nil {
			log.Errorf("error starting game: %v", err)
		}

	case CommandCancel:
		game.ManagerInstance(session).EndGame(ic.ChannelID)
		session.ChannelMessageSend(ic.ChannelID, "District uprising ended the games early. The dissidents of the uprising will be eliminated.")

	case CommandClear:
		session.ChannelMessageSend(ic.ChannelID, "This command is not yet supported. Panem issues our sincerest apologies and politely requests your obedience.")
	}
}
