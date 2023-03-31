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
	log "github.com/sirupsen/logrus"
)

func benchmarkGameRun(b *testing.B, cfg GameConfig, members []*discordgo.Member) {
	b.Helper()

	sender := &BufferSender{SendLatency: 100 * time.Millisecond}

	cfg.Sender = sender
	g := NewGame(cfg)
	g.introMessage = &discordgo.Message{ID: "123"}

	for i := 0; i < len(members); i++ {
		g.RegisterUser("123", settings.ParticipantEmojiName, NewParticipant(members[i]))
	}

	g.run(context.Background())
}

func benchmarkSetup(b *testing.B, userCount, multiplier int) (PhraseGenerator, []*discordgo.Member) {
	b.Helper()
	log.SetLevel(log.WarnLevel)
	data, err := os.ReadFile(path.Join("..", settings.DataLocation, settings.PhrasesFile))
	if err != nil {
		b.Fatal(err)
	}

	var members []*discordgo.Member
	for j := 0; j < userCount; j++ {
		for k := 0; k < multiplier; k++ {
			members = append(members, &discordgo.Member{
				User: &discordgo.User{
					ID:       fmt.Sprintf("%v-%v", j, k),
					Username: fmt.Sprintf("user-%v-%v", j, k),
				},
			})
		}
	}

	return lib.NewJSONPhrases(data), members
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
	}

	for name, test := range tests {
		b.Run(name, func(b *testing.B) {
			jp, users := benchmarkSetup(b, test.UserCount, test.EntryMultiplier)

			for i := 0; i < b.N; i++ {
				benchmarkGameRun(
					b,
					GameConfig{
						ChannelID:       "123",
						DayDelay:        1 * time.Nanosecond,
						PhraseGenerator: jp,
						Session:         &discordgo.Session{},
						Sponsor:         "Sponsor",
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
	SendLatency time.Duration
	sync.Mutex
}

func (b *BufferSender) SendNormal(str string) (*discordgo.Message, error) {
	return b.send(str)
}

func (b *BufferSender) SendEmbed(str string) (*discordgo.Message, error) {
	return b.send(str)
}

func (b *BufferSender) send(str string) (*discordgo.Message, error) {
	b.Lock()
	defer b.Unlock()
	b.buffer = append(b.buffer, str)
	time.Sleep(b.SendLatency)
	return nil, nil
}

func (b *BufferSender) ClearBuffer() {
	b.buffer = nil
}
