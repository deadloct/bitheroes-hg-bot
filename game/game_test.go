package game

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/lib"
	"github.com/deadloct/bitheroes-hg-bot/settings"
)

func benchmarkGameRun(b *testing.B, cfg GameConfig, users []*discordgo.User) {
	b.Helper()

	sender := &BufferSender{SendLatency: 100 * time.Millisecond}
	msgChan := sender.Start()
	defer sender.Stop()

	cfg.Sender = sender
	g := NewGame(cfg)

	// Hacks to avoid using g.Start, which waits for reactions
	g.sendCh = msgChan

	g.run(context.Background())
}

func benchmarkSetup(b *testing.B, userCount, multiplier int) (PhraseGenerator, []*discordgo.User) {
	b.Helper()
	data, err := os.ReadFile(path.Join("..", settings.DataLocation, settings.PhrasesFile))
	if err != nil {
		b.Fatal(err)
	}

	var users []*discordgo.User
	for j := 0; j < userCount; j++ {
		for k := 0; k < multiplier; k++ {
			users = append(users, &discordgo.User{
				ID:       fmt.Sprintf("%v-%v", j, k),
				Username: fmt.Sprintf("user-%v-%v", j, k),
			})
		}
	}

	return lib.NewJSONPhrases(data), users
}

func BenchmarkGameDuration(b *testing.B) {
	tests := map[string]struct {
		UserCount       int
		EntryMultiplier int
	}{
		"1 user, 1 multiplier": {
			UserCount:       1,
			EntryMultiplier: 1,
		},
		"100 users, 1 multiplier": {
			UserCount:       100,
			EntryMultiplier: 1,
		},
		"1 user, 100 multiplier": {
			UserCount:       1,
			EntryMultiplier: 100,
		},
		"100 users, 100 multiplier": {
			UserCount:       100,
			EntryMultiplier: 100,
		},
		"1000 users, 1 multiplier": {
			UserCount:       1000,
			EntryMultiplier: 1,
		},
		"1000 users, 100 multiplier": {
			UserCount:       1000,
			EntryMultiplier: 100,
		},
	}

	for name, test := range tests {
		b.Run(name, func(b *testing.B) {
			jp, users := benchmarkSetup(b, test.UserCount, test.EntryMultiplier)

			for i := 0; i < b.N; i++ {
				benchmarkGameRun(
					b,
					GameConfig{
						Author:          &discordgo.User{Username: "sponsor", ID: "123"},
						DayDelay:        1 * time.Nanosecond,
						PhraseGenerator: jp,
						Session:         &discordgo.Session{},
						EntryMultiplier: test.EntryMultiplier,
						VictorCount:     1,
					},
					users,
				)
			}
		})
	}
}

type BufferSender struct {
	buffer      []string
	msgCh       chan string
	stopCh      chan struct{}
	SendLatency time.Duration
	sync.Mutex
}

func (b *BufferSender) Start() chan string {
	b.msgCh = make(chan string)
	b.stopCh = make(chan struct{})
	go b.listen()
	return b.msgCh
}

func (b *BufferSender) listen() {
	for {
		select {
		case msg := <-b.msgCh:
			b.Lock()
			if msg != "" {
				b.buffer = append(b.buffer, msg)
			}
			b.Unlock()
		case <-b.stopCh:
			return
		}
	}
}

func (b *BufferSender) Stop() {
	close(b.msgCh)
	close(b.stopCh)
}

func (b *BufferSender) Send(str string) (*discordgo.Message, error) {
	b.Lock()
	defer b.Unlock()
	b.buffer = append(b.buffer, str)
	time.Sleep(b.SendLatency)
	return nil, nil
}

func (b *BufferSender) ClearBuffer() {
	b.buffer = nil
}
