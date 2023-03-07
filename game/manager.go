package game

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type RunningGame struct {
	Game   *Game
	StopCh chan struct{}
}

type Manager struct {
	games   map[string]*RunningGame // maps channel ID to games; one allowed per channel at a time
	session *discordgo.Session
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

func (m *Manager) StartGame(channel, cmd string, author *discordgo.User) error {
	log.Infof("starting game in channel %v", channel)
	sender := NewDiscordSender(m.session, channel)
	sender.Start()

	if _, exists := m.games[channel]; exists {
		log.Errorf("game already exists in channel %v", channel)
		sender.Send("There is already a running game in this channel.")
		sender.Stop()
		return fmt.Errorf("game already exists in channel %s", channel)
	}

	g := NewGame(m.session, author, cmd, sender)

	doneCh, err := g.Start()
	if err != nil {
		log.Errorf("error starting game: %v", err)
		sender.Send("There was an unexpected error starting a game.")
		sender.Stop()
		return err
	}

	m.games[channel] = &RunningGame{
		Game:   g,
		StopCh: doneCh,
	}
	return nil
}

func (m *Manager) EndGame(channel string) {
	if rg, exists := m.games[channel]; exists {
		log.Infof("ending game in channel %v", channel)
		rg.StopCh <- struct{}{}
		delete(m.games, channel)
	}
}

func (m *Manager) ClearMessagesForChannel(channel string) {
	if _, exists := m.games[channel]; exists {
		log.Info("clearing messages in channel %v", channel)
		// TODO: Implement this
		// rg.Game.ClearMessages()
	}
}
