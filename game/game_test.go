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

func testSetupGameRun(f Fataler, userCount, multiplier int) (PhraseGenerator, []*discordgo.Member) {
	f.Helper()
	log.SetLevel(log.WarnLevel)
	data, err := os.ReadFile(path.Join("..", settings.DataLocation, settings.PhrasesFile))
	if err != nil {
		f.Fatal(err)
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

func testRunGame(f Fataler, cfg GameConfig, members []*discordgo.Member) []*Participant {
	f.Helper()

	sender := &BufferSender{SendLatency: 100 * time.Millisecond}

	cfg.Sender = sender
	g := NewGame(cfg)
	g.introMessage = &discordgo.Message{ID: "123"}

	emoji := settings.GetEmoji(settings.EmojiParticipant).Name
	for i := 0; i < len(members); i++ {
		g.RegisterUser("123", emoji, NewParticipant(members[i]))
	}

	return g.run(context.Background())
}

func TestGame_Run(t *testing.T) {
	sender := &BufferSender{SendLatency: 100 * time.Millisecond}

	tests := map[string]struct {
		UserCount int
		Clone     int
		Victors   int
		RunCount  int
	}{
		"1 user, no clones, 1 victors": {
			UserCount: 1,
			Clone:     1,
			Victors:   1,
			RunCount:  5,
		},
		"100 users, no clones, 1 victor": {
			UserCount: 100,
			Clone:     1,
			Victors:   1,
			RunCount:  5,
		},
		"100 users, no clones, 10 victors": {
			UserCount: 100,
			Clone:     1,
			Victors:   10,
			RunCount:  5,
		},
		"100 users, 5 clones, 5 victors": {
			UserCount: 100,
			Clone:     5,
			Victors:   5,
			RunCount:  1,
		},
		"20 users, no clones, 19 victors": {
			UserCount: 20,
			Clone:     1,
			Victors:   19,
			RunCount:  10,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < test.RunCount; i++ {
				jp, users := testSetupGameRun(t, test.UserCount, test.Clone)

				cfg := GameConfig{
					Channel:         &discordgo.Channel{ID: "123", Name: "123"},
					Guild:           &discordgo.Guild{ID: "123", Name: "123"},
					DayDelay:        1 * time.Nanosecond,
					PhraseGenerator: jp,
					Session:         &discordgo.Session{},
					Sponsor:         "Sponsor",
					Clone:           test.Clone,
					StartedBy: NewParticipant(&discordgo.Member{
						User: &discordgo.User{
							ID:       "123",
							Username: "Hello",
						},
					}),
					VictorCount: test.Victors,
				}

				cfg.Sender = sender
				g := NewGame(cfg)
				g.introMessage = &discordgo.Message{ID: "123"}

				victors := testRunGame(t, cfg, users)

				if len(victors) != test.Victors {
					t.Fatalf("expected %v victors but got %v", test.Victors, len(victors))
				}
			}
		})
	}

}

type Fataler interface {
	Helper()
	Fatal(args ...any)
}

func BenchmarkGameDuration(b *testing.B) {
	tests := map[string]struct {
		UserCount int
		Clone     int
	}{
		"1 user, 1 multiplier": {
			UserCount: 1,
			Clone:     1,
		},
		"100 users, 1 multiplier": {
			UserCount: 100,
			Clone:     1,
		},
		"1 user, 100 multiplier": {
			UserCount: 1,
			Clone:     100,
		},
		"100 users, 100 multiplier": {
			UserCount: 100,
			Clone:     100,
		},
		"1000 users, 1 multiplier": {
			UserCount: 1000,
			Clone:     1,
		},
	}

	for name, test := range tests {
		b.Run(name, func(b *testing.B) {
			jp, users := testSetupGameRun(b, test.UserCount, test.Clone)

			for i := 0; i < b.N; i++ {
				testRunGame(
					b,
					GameConfig{
						Channel:         &discordgo.Channel{ID: "123", Name: "123"},
						Guild:           &discordgo.Guild{ID: "123", Name: "123"},
						DayDelay:        1 * time.Nanosecond,
						PhraseGenerator: jp,
						Session:         &discordgo.Session{},
						Sponsor:         "Sponsor",
						Clone:           test.Clone,
						StartedBy: NewParticipant(&discordgo.Member{
							User: &discordgo.User{
								ID:       "123",
								Username: "Hello",
							},
						}),
						VictorCount: 1,
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

func (b *BufferSender) Send(str string) (*discordgo.Message, error) {
	return b.send(str)
}

func (b *BufferSender) SendQuoted(str string) (*discordgo.Message, error) {
	return b.send(str)
}

func (b *BufferSender) SendEmbed(str string) (*discordgo.Message, error) {
	return b.send(str)
}

func (b *BufferSender) SendDM(user *discordgo.User, msg string) error {
	return nil
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
