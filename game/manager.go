package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type RunningGame struct {
	Game   *Game
	Cancel context.CancelFunc
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

func (m *Manager) StartGame(channel string, delay time.Duration, author *discordgo.User) error {
	sender := NewDiscordSender(m.session, channel)
	sender.Start()

	if !m.CanStart(channel) {
		log.Errorf("game already running in channel %v", channel)
		sender.Send("There is already an active Hunger Games running in this channel, please wait for it to finish or stop the existing game first.")
		sender.Stop()
		return fmt.Errorf("game already exists in channel %s", channel)
	}

	sender.Send("Starting a Hunger Games event in this channel. Any existing, unstarted games will be cancelled.")
	log.Infof("starting game in channel %v", channel)

	g := NewGame(GameConfig{
		Author:  author,
		Delay:   delay,
		Sender:  sender,
		Session: m.session,
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
	m.games[channel] = &RunningGame{
		Game:   g,
		Cancel: cancel,
	}
	m.Unlock()

	return nil
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

func (m *Manager) ClearMessagesForChannel(channel string) {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.games[channel]; exists {
		log.Info("clearing messages in channel %v", channel)
		// TODO: Implement this
		// rg.Game.ClearMessages()
	}
}
