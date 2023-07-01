package cmd

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/game"
	"github.com/deadloct/bitheroes-hg-bot/lib"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

const (
	CommandPrefix                 = "hg-"
	CommandHelp                   = CommandPrefix + "help"
	CommandStart                  = CommandPrefix + "start"
	CommandStartOptionClone       = "clone"
	CommandStartOptionNotify      = "notify"
	CommandStartOptionMinimumTier = "minimum-tier"
	CommandStartOptionSponsor     = "sponsor"
	CommandStartOptionStartDelay  = "start-delay"
	CommandStartOptionVictorCount = "victors"
	CommandCancel                 = CommandPrefix + "cancel"
	CommandClear                  = CommandPrefix + "clear"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^\p{L}\p{N}-_\.\[\] ]+`)

	CommandStartOptionMinimumTierMinValue float64 = 2
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
				Type: discordgo.ApplicationCommandOptionInteger,
				Name: CommandStartOptionStartDelay,
				Description: fmt.Sprintf(
					"Seconds to wait for reactions before starting game. Default: %v, Min: %v, Max: %v (%v)",
					settings.DefaultStartDelay,
					settings.MinimumStartDelay,
					settings.MaximumStartDelay,
					time.Duration(settings.MaximumStartDelay)*time.Second,
				),
				Required: false,
			},
			{
				Type: discordgo.ApplicationCommandOptionInteger,
				Name: CommandStartOptionVictorCount,
				Description: fmt.Sprintf(
					"Number of victors (winners). Default: %v, Min: %v",
					settings.DefaultVictorCount, settings.DefaultVictorCount),
				Required: false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        CommandStartOptionSponsor,
				Description: "The sponsor of the event. Default: user running this command",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        CommandStartOptionNotify,
				Description: "User to notify about result",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        CommandStartOptionMinimumTier,
				Description: "Minimum tier of contestants",
				Required:    false,
				MinValue:    &CommandStartOptionMinimumTierMinValue,
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

func init() {
	// Debug option to amplify entries
	if settings.EnableClone {
		for _, command := range commands {
			if command.Name == CommandStart {
				command.Options = append(command.Options, &discordgo.ApplicationCommandOption{
					Type: discordgo.ApplicationCommandOptionInteger,
					Name: CommandStartOptionClone,
					Description: fmt.Sprintf(
						"Number of entries per tribute. Default: %v, Min: %v, Max: %v",
						settings.DefaultClone, settings.MinimumClone, settings.MaximumClone),
					Required: false,
				})
			}
		}
	}
}

type Manager struct {
	phraseData []byte
	jokeData   []byte
}

func NewManager(phraseData []byte, jokeData []byte) *Manager {
	return &Manager{phraseData: phraseData, jokeData: jokeData}
}

func (m *Manager) RegisterCommands(session *discordgo.Session) error {
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

func (m *Manager) DeregisterCommmands(session *discordgo.Session) error {
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

func (m *Manager) CommandHandler(session *discordgo.Session, ic *discordgo.InteractionCreate) {
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
		Data: &discordgo.InteractionResponseData{Content: "> Command acknowledged. Engaging the capitol of Panem."},
	})

	options := ic.ApplicationCommandData().Options

	startedBy := game.NewParticipant(ic.Member)
	log.Infof("%v issued command %v", startedBy.DisplayFullName(), ic.ApplicationCommandData().Name)

	v := ic.ApplicationCommandData().Name
	switch v {
	case CommandHelp:
		session.ChannelMessageSend(ic.ChannelID, settings.Help)

	case CommandStart:
		var minimumTier int
		var notify *discordgo.User

		delay := settings.DefaultStartDelay * time.Second
		clone := settings.DefaultClone
		victors := settings.DefaultVictorCount
		sponsor := startedBy.DisplayName()

		for _, option := range options {
			switch option.Name {
			case CommandStartOptionStartDelay:
				v := int(option.IntValue())
				switch {
				case v < settings.MinimumStartDelay:
					msg := fmt.Sprintf("> The delay of %v is much too short. Hunger Games will wait for %v seconds instead.", v, settings.DefaultStartDelay)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				case v > settings.MaximumStartDelay:
					msg := fmt.Sprintf("> The delay of %v is much too long. Hunger Games will wait for %v seconds instead.", v, settings.DefaultStartDelay)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				default:
					delay = time.Duration(v) * time.Second
				}

			case CommandStartOptionClone:
				v := int(option.IntValue())
				switch {
				case v < settings.MinimumClone:
					clone = settings.MinimumClone
					msg := fmt.Sprintf("> The multiplier of %v is much too low. Setting to %v instead.", v, settings.MinimumClone)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				case v > settings.MaximumClone:
					clone = settings.MaximumClone
					msg := fmt.Sprintf("> The multiplier of %v is much too high. Setting to %v instead.", v, settings.MaximumClone)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				default:
					clone = v
				}

			case CommandStartOptionVictorCount:
				v := int(option.IntValue())
				switch {
				case v < settings.MinimumVictorCount:
					victors = settings.DefaultVictorCount
					msg := fmt.Sprintf("> Victors of %v is much too low. Setting to %v instead.", v, settings.DefaultVictorCount)
					session.ChannelMessageSend(ic.ChannelID, msg)
					log.Warn(msg)
				default:
					victors = v
				}

			case CommandStartOptionSponsor:
				v := option.StringValue()
				if v != "" {
					sponsor = m.sanitize(v)
				}

			case CommandStartOptionNotify:
				v := option.UserValue(session)
				if v != nil {
					notify = v
				}

			case CommandStartOptionMinimumTier:
				v := int(option.IntValue())
				if v > 1 {
					minimumTier = v
				}
			}
		}

		if victors == 0 {
			session.ChannelMessageSend(ic.ChannelID, "> There will be no victors this year. An uprising broke out in the underground Bit Heroes sector, but rest easy knowing that the dissidents of the uprising will be eliminated.")
			return
		}

		jp := lib.NewJSONPhrases(m.phraseData)
		log.Infof("imported %v phrases", jp.PhraseCount())

		jj, err := lib.NewJSONJokes(m.jokeData)
		if err != nil {
			log.Warnf("unable to load jokes: %v", err)
		}

		cfg := game.GameStartConfig{
			Channel:         ic.ChannelID,
			Delay:           delay,
			Clone:           clone,
			MinimumTier:     minimumTier,
			Notify:          notify,
			JokeGenerator:   jj,
			PhraseGenerator: jp,
			Sponsor:         sponsor,
			StartedBy:       startedBy,
			VictorCount:     victors,
		}

		if err := game.ManagerInstance(session).StartGame(cfg); err != nil {
			log.Errorf("error starting game: %v", err)
		}

	case CommandCancel:
		game.ManagerInstance(session).EndGame(ic.ChannelID)
		session.ChannelMessageSend(ic.ChannelID, "> District uprising ended the games early. The dissidents of the uprising will be eliminated.")

	case CommandClear:
		game.ManagerInstance(session).ClearBotMessages(ic.ChannelID)
	}
}

func (m *Manager) sanitize(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}
