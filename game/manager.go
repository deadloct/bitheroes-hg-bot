package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

type RunningGame struct {
	Game   *Game
	Cancel context.CancelFunc
}

type GameStartConfig struct {
	Channel         *discordgo.Channel
	Guild           *discordgo.Guild
	Delay           time.Duration
	Clone           int
	JokeGenerator   JokeGenerator
	MinimumTier     int
	Notify          *discordgo.User
	PhraseGenerator PhraseGenerator
	Sponsor         string
	StartedBy       *Participant
	VictorCount     int
}

type Manager struct {
	games   map[string]*RunningGame // maps channel ID to games; one allowed per channel at a time
	session *discordgo.Session
	sync.Mutex
}

var (
	managerSingleton     *Manager
	managerSingletonOnce sync.Once
)

func ManagerInstance(session *discordgo.Session) *Manager {
	managerSingletonOnce.Do(func() {
		managerSingleton = &Manager{
			games:   make(map[string]*RunningGame),
			session: session,
		}
	})

	return managerSingleton
}

func (m *Manager) StartGame(cfg GameStartConfig) error {
	if cfg.Channel == nil {
		return errors.New("no access to this channel")
	}

	sender := NewDiscordSender(m.session, cfg.Channel.ID)

	if !m.CanStart(cfg.Channel.ID) {
		log.Errorf("game already running in channel %v", cfg.Channel.Name)
		sender.SendQuoted("There is already an active Hunger Games running in this channel, please wait for it to finish or stop the existing game first.")
		return fmt.Errorf("game already exists in channel %s", cfg.Channel.Name)
	}

	sender.SendQuoted("Starting a Hunger Games event in this channel.")
	m.EndGame(cfg.Channel.ID)

	log.Infof("%v started a game channel:%v server:%v config:%#v", cfg.StartedBy.DisplayFullName(), cfg.Channel.Name, cfg.Guild.Name, cfg)

	g := NewGame(GameConfig{
		Delay:           cfg.Delay,
		Guild:           cfg.Guild,
		Channel:         cfg.Channel,
		Clone:           cfg.Clone,
		JokeGenerator:   cfg.JokeGenerator,
		MinimumTier:     cfg.MinimumTier,
		Notify:          cfg.Notify,
		PhraseGenerator: cfg.PhraseGenerator,
		Sender:          sender,
		Session:         m.session,
		Sponsor:         cfg.Sponsor,
		StartedBy:       cfg.StartedBy,
		VictorCount:     cfg.VictorCount,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := g.Start(ctx)
	if err != nil {
		log.Errorf("error starting game: %v", err)
		sender.SendQuoted("There was an unexpected error starting the game.")
		cancel()
		return err
	}

	m.Lock()
	m.games[cfg.Channel.ID] = &RunningGame{
		Game:   g,
		Cancel: cancel,
	}
	m.Unlock()

	return nil
}

func (m *Manager) ReactionHandler(session *discordgo.Session, mra *discordgo.MessageReactionAdd) {
	m.Lock()
	defer m.Unlock()

	rg, ok := m.games[mra.ChannelID]
	if !ok {
		return
	}

	rg.Game.RegisterUser(mra.MessageID, mra.Emoji.Name, NewParticipant(mra.Member))
}

func (m *Manager) EndGame(channel string) {
	m.Lock()
	defer m.Unlock()

	if rg, exists := m.games[channel]; exists {
		log.Infof("ending game in channel %v", channel)
		rg.Cancel()
		delete(m.games, channel)
	}
}

func (m *Manager) CanStart(channel string) bool {
	m.Lock()
	defer m.Unlock()

	if rg, exists := m.games[channel]; exists && rg.Game.IsRunning() {
		return false
	}

	return true
}

func (m *Manager) RetrieveBotMessagesInChannel(channelID, lastMessageID string, messages []string) ([]string, error) {
	new, err := m.session.ChannelMessages(channelID, settings.DiscordMaxMessages, lastMessageID, "", "")
	if err != nil {
		log.Errorf("could not retrieve messages: %v", err)
		return nil, err
	}

	for i := 0; i < len(new); i++ {
		// Only return bot messages
		if new[i].Author.ID == m.session.State.User.ID {
			messages = append(messages, new[i].ID)
		}
	}

	if len(new) == settings.DiscordMaxMessages {
		last := new[len(new)-1].ID
		return m.RetrieveBotMessagesInChannel(channelID, last, messages)
	}

	return messages, nil
}

func (m *Manager) ClearBotMessages(channelID string) error {
	log.Infof("attempting to delete bot messages in channel %v", channelID)

	msgs, err := m.RetrieveBotMessagesInChannel(channelID, "", nil)
	if err != nil {
		log.Errorf("unable to retrieve messages to delete: %v", err)
	}

	log.Infof("deleting %v bot messages in channel %v", len(msgs), channelID)

	if len(msgs) > 0 {
		return m.clearBotMessagesRecursively(channelID, msgs)
	}

	log.Info("finished deleting messages")
	return nil
}

func (m *Manager) clearBotMessagesRecursively(channelID string, messages []string) error {
	var toDelete []string
	if len(toDelete) > 0 {
		toDelete = messages[:settings.DiscordMaxBulkDelete]
	} else {
		toDelete = messages
	}

	if err := m.session.ChannelMessagesBulkDelete(channelID, toDelete); err != nil {
		log.Errorf("could not bulk delete messages: %v", err)
		return err
	}

	if len(messages) > settings.DiscordMaxBulkDelete {
		messages = messages[settings.DiscordMaxBulkDelete:]
		return m.clearBotMessagesRecursively(channelID, messages)
	}

	return nil
}
