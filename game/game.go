package game

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/deadloct/bitheroes-hg-bot/lib"
	"github.com/deadloct/bitheroes-hg-bot/settings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type GameState int

const (
	NotStarted GameState = iota
	Started
	Finished
	Cancelled
)

type PhraseGenerator interface {
	GetRandomPhrase(user, mention string, alive []string) string
}

type JokeGenerator interface {
	GetJoke() (lib.Joke, error)
}

type GameConfig struct {
	Author          *Participant
	ChannelID       string
	DayDelay        time.Duration
	Delay           time.Duration // delayed start
	EntryMultiplier int
	JokeGenerator   JokeGenerator
	PhraseGenerator PhraseGenerator
	Sender          Sender
	Session         *discordgo.Session
	VictorCount     int
}

type Game struct {
	GameConfig

	channelName    string
	introMessage   *discordgo.Message
	serverName     string
	state          GameState
	participants   []*Participant
	participantMap map[string]*Participant

	sync.Mutex
}

func NewGame(cfg GameConfig) *Game {
	if cfg.DayDelay == 0 {
		cfg.DayDelay = settings.DefaultDayDelay
	}

	channelName := cfg.ChannelID
	serverName := "unknown"
	if channel, err := cfg.Session.State.Channel(cfg.ChannelID); err == nil {
		channelName = channel.Name
		if server, err := cfg.Session.State.Guild(channel.GuildID); err != nil {
			serverName = server.Name
		}
	}

	return &Game{
		GameConfig:     cfg,
		participantMap: make(map[string]*Participant),
		channelName:    channelName,
		serverName:     serverName,
	}
}

func (g *Game) Start(ctx context.Context) error {
	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(settings.IntroValues{
		Delay:       g.Delay,
		EmojiCode:   settings.ParticipantEmojiCode,
		User:        g.Author.DisplayName(),
		VictorCount: g.VictorCount,
	})
	if err != nil {
		return err
	}

	g.logMessage(log.DebugLevel, "sending intro")
	if g.introMessage, err = g.Sender.SendEmbed(intro); err != nil {
		return err
	}

	g.Session.MessageReactionAdd(g.introMessage.ChannelID, g.introMessage.ID,
		fmt.Sprintf("%v:%v", settings.ParticipantEmojiName, settings.ParticipantEmojiID))

	g.delayedStart(ctx)
	return nil
}

func (g *Game) HasStarted() bool {
	g.Lock()
	defer g.Unlock()

	return g.state != NotStarted
}

func (g *Game) IsRunning() bool {
	g.Lock()
	defer g.Unlock()

	return g.state == Started
}

func (g *Game) RegisterUser(messageID, emoji string, participant *Participant) {
	if g.HasStarted() {
		return
	}

	if emoji != settings.ParticipantEmojiName {
		return
	}

	if messageID != g.introMessage.ID {
		return
	}

	if participant.User.Bot {
		return
	}

	if _, ok := g.participantMap[participant.User.ID]; !ok {
		g.participantMap[participant.User.ID] = participant
		g.participants = append(g.participants, participant)
	}
}

func (g *Game) getIntro(vals settings.IntroValues) (string, error) {
	var result bytes.Buffer
	if err := settings.Intro.Execute(&result, vals); err != nil {
		return "", err
	}

	return result.String(), nil
}

func (g *Game) delayedStart(ctx context.Context) {
	g.logMessage(log.DebugLevel, "delaying start by %v", g.Delay)

	jokeCh := make(chan struct{})
	NewJester(g.JokeGenerator, g.Sender, g.Session).StartRandomJokes(ctx, jokeCh)

	go func() {
		select {
		case <-ctx.Done():
			jokeCh <- struct{}{}
			g.logMessage(log.InfoLevel, "context done, cancelling game")
			g.Lock()
			g.state = Cancelled
			g.Unlock()

		case <-time.After(g.Delay):
			jokeCh <- struct{}{}
			g.run(ctx)
		}
	}()
}

func (g *Game) run(ctx context.Context) {
	g.logMessage(log.InfoLevel, "starting game with user count %v", len(g.participants))
	g.Lock()
	g.state = Started
	g.Unlock()

	if len(g.participants) == 0 {
		g.logMessage(log.InfoLevel, "no users entered")
		g.Sender.SendNormal(fmt.Sprintf("No tributes have come forward within %v. This district will be eliminated.", g.Delay))
		g.Lock()
		g.state = Cancelled
		g.Unlock()
		return
	}

	if g.EntryMultiplier > 1 {
		for _, p := range g.participants {
			for i := 2; i <= g.EntryMultiplier; i++ {
				name := fmt.Sprintf("%v-%v", p.DisplayName(), i)
				g.participants = append(g.participants, NewParticipant(&discordgo.Member{
					Nick: name,
					User: &discordgo.User{
						Username:      name,
						ID:            p.User.ID,
						Discriminator: p.User.Discriminator,
					},
				}))
			}
		}
	}

	g.sendTributeOutput(g.participants)

	for day := 0; len(g.participants) > g.VictorCount; day++ {
		time.Sleep(g.DayDelay)

		select {
		case <-ctx.Done():
			g.logMessage(log.InfoLevel, "context done, cancelling game on day %v", day)
			g.Lock()
			g.state = Cancelled
			g.Unlock()
			return

		default:
			g.logMessage(log.InfoLevel, "simulating day %v with %v tributes", day, len(g.participants))
			var err error
			g.participants, err = g.runDay(ctx, day, g.participants)
			if err != nil {
				g.logMessage(log.ErrorLevel, "failed to simulate day %v: %v", day, err)
				g.Sender.SendNormal(fmt.Sprintf("failed to run game for day %v", day+1))
				g.Lock()
				g.state = Cancelled
				g.Unlock()
				return
			}

			g.logMessage(log.InfoLevel, "users left after day %v: %v", day, len(g.participants))
		}
	}

	var mentions []string
	for _, p := range g.participants {
		mentions = append(mentions, p.Mention())
		g.logMessage(log.InfoLevel, "winner: %v %v#%v %v", p.DisplayName(), p.User.Username, p.User.Discriminator, p.User.ID)
	}

	mentionStr := strings.Join(mentions, ", ")
	g.sendBatchOutput([]string{
		"**This year's Hunger Games have concluded**",
		settings.WhiteSpaceChar,
		fmt.Sprintf("%v: %v", "Congratulations to our new victor(s)", mentionStr),
	})

	g.Lock()
	g.state = Finished
	g.Unlock()
}

func (g *Game) runDay(ctx context.Context, day int, participants []*Participant) ([]*Participant, error) {
	if len(participants) == 1 {
		return participants, nil
	}

	output := []string{
		fmt.Sprintf(
			":%v:   **DAY %v**   :%v:",
			settings.DayEmoji,
			day+1,
			settings.DayEmoji,
		),
		settings.WhiteSpaceChar,
	}

	// min and max are 0-based
	var min int
	max := (len(participants) / 2) + 1 // +1 b/c right is exclusive

	// 1/2 - 3/4 on the first day, it's always a slaughter
	if day == 0 && len(participants) > 5 {
		min = max / 2
		max = max * 3 / 4
	}

	killCount, err := lib.GetRandomInt(min, max)
	if err != nil {
		g.logMessage(log.ErrorLevel, "failed to get random number between %v and %v: %v", min, max, err)
		return nil, err
	}

	if killCount == 0 {
		output = append(output, fmt.Sprintf("All was quiet on day %v.", day+1))
		g.sendBatchOutput(output)
		return participants, nil
	}

	dead := make(map[int]struct{})
	for i := 0; i < killCount; i++ {
		for {
			toDie, err := lib.GetRandomInt(0, len(participants))
			if err != nil {
				g.logMessage(log.ErrorLevel, "failed to get random number between %v and %v: %v", min, max, err)
				return nil, err
			}

			if _, alreadyDead := dead[toDie]; !alreadyDead {
				dead[toDie] = struct{}{}
				break
			}
		}
	}

	var living []*Participant
	var livingNames []string
	for i, p := range participants {
		if _, dead := dead[i]; !dead {
			living = append(living, p)
			livingNames = append(livingNames, p.DisplayName())
		}
	}

	for i := range dead {
		mention := ""
		if g.EntryMultiplier == 1 {
			mention = participants[i].Mention()
		}

		line := "• " + g.PhraseGenerator.GetRandomPhrase(participants[i].DisplayName(), mention, livingNames)
		g.logMessage(log.DebugLevel, "Day %v: %v", day, line)
		output = append(output, line)
	}

	output = append(output, settings.WhiteSpaceChar, fmt.Sprintf(
		"%v player(s) remain at the end of day %v: %v",
		len(living),
		day+1,
		strings.Join(livingNames, ", "),
	))

	select {
	case <-ctx.Done():
	default:
		g.sendBatchOutput(output)
	}

	return living, nil
}

func (g *Game) sendTributeOutput(participants []*Participant) {
	g.logMessage(log.DebugLevel, "tribute count: %v", len(participants))

	tributeLines := []string{
		"**Please welcome our brave tributes!**",
		settings.WhiteSpaceChar,
		"What a fantastic group of individuals we have for this year's contest:",
	}

	var tributes []string
	for _, p := range participants {
		tributes = append(tributes, p.DisplayName())
	}

	tributeLines = append(tributeLines, strings.Join(tributes, ", "))
	g.sendBatchOutput(tributeLines)
}

func (g *Game) sendBatchOutput(lines []string) {
	g.Sender.SendNormal(strings.Join(lines, "\n"))
}

func (g *Game) logMessage(level log.Level, msg string, args ...interface{}) {
	suffix := fmt.Sprintf(" (server:'%s' channel:'%s')", g.serverName, g.channelName)
	switch level {
	case log.TraceLevel:
		log.Tracef(msg+suffix, args...)
	case log.DebugLevel:
		log.Debugf(msg+suffix, args...)
	case log.InfoLevel:
		log.Infof(msg+suffix, args...)
	case log.WarnLevel:
		log.Warnf(msg+suffix, args...)
	case log.ErrorLevel:
		log.Errorf(msg+suffix, args...)
	case log.PanicLevel:
		log.Panicf(msg+suffix, args...)
	}
}
