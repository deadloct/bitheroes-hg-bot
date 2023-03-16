package game

import (
	"context"
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
	Author          *discordgo.User
	Channel         string
	Delay           time.Duration
	PhraseGenerator PhraseGenerator
	EntryMultiplier int
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
	sender := NewDiscordSender(m.session, cfg.Channel)
	sender.Start()

	if !m.CanStart(cfg.Channel) {
		log.Errorf("game already running in channel %v", cfg.Channel)
		sender.Send("There is already an active Hunger Games running in this channel, please wait for it to finish or stop the existing game first.")
		sender.Stop()
		return fmt.Errorf("game already exists in channel %s", cfg.Channel)
	}

	sender.Send("Starting a Hunger Games event in this channel. Any existing, unstarted games will be cancelled.")
	m.EndGame(cfg.Channel)

	log.Infof("starting game in channel %v", cfg.Channel)

	g := NewGame(GameConfig{
		Author:          cfg.Author,
		Delay:           cfg.Delay,
		EntryMultiplier: cfg.EntryMultiplier,
		PhraseGenerator: cfg.PhraseGenerator,
		Sender:          sender,
		Session:         m.session,
		VictorCount:     cfg.VictorCount,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := g.Start(ctx)
	if err != nil {
		log.Errorf("error starting game: %v", err)
		sender.Send("There was an unexpected error starting the game.")
		sender.Stop()
		cancel()
		return err
	}

	m.Lock()
	m.games[cfg.Channel] = &RunningGame{
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

	rg.Game.RegisterUser(mra.MessageID, mra.Emoji.Name, mra.Member.User)
}

func (m *Manager) EndGame(channel string) {
	m.Lock()
	defer m.Unlock()

	if rg, exists := m.games[channel]; exists {
		log.Infof("ending game in channel %v", channel)
		rg.Cancel()
		rg.Game.Sender.Stop()
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
		// Only delete bot messages
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
