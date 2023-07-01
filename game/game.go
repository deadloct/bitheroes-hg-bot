package game

import (
	"bytes"
	"context"
	"fmt"
	"math"
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
	ChannelID       string
	DayDelay        time.Duration
	Delay           time.Duration // delayed start
	Clone           int
	JokeGenerator   JokeGenerator
	MinimumTier     int
	Notify          *discordgo.User
	PhraseGenerator PhraseGenerator
	Sender          Sender
	Session         *discordgo.Session
	Sponsor         string
	StartedBy       *Participant
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
	participantEmoji := settings.GetEmoji(settings.EmojiParticipant)
	effieEmoji := settings.GetEmoji(settings.EmojiEffie)

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(settings.IntroValues{
		Delay:       g.Delay,
		EntryEmoji:  participantEmoji.EmojiCode(),
		EffieEmoji:  effieEmoji.EmojiCode(),
		MinimumTier: g.MinimumTier,
		Sponsor:     g.Sponsor,
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
		fmt.Sprintf("%v:%v", participantEmoji.Name, participantEmoji.ID))

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

	// Not Started is a waiting game, so it counts as "running"
	return g.state == Started || g.state == NotStarted
}

func (g *Game) RegisterUser(messageID, emoji string, participant *Participant) {
	if g.HasStarted() {
		return
	}

	participantEmoji := settings.GetEmoji(settings.EmojiParticipant)
	if emoji != participantEmoji.Name {
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

func (g *Game) run(ctx context.Context) []*Participant {
	g.logMessage(log.InfoLevel, "starting game with user count %v", len(g.participants))
	g.Lock()
	g.state = Started
	g.Unlock()

	if len(g.participants) == 0 {
		g.logMessage(log.InfoLevel, "no users entered")
		g.Sender.SendQuoted(fmt.Sprintf("No tributes have come forward within %v. This district will be eliminated.", g.Delay))
		g.Lock()
		g.state = Cancelled
		g.Unlock()
		return nil
	}

	g.sendTributeOutput(g.participants)

	// Clone tributes
	if g.Clone > 1 {
		for _, p := range g.participants {
			for i := 2; i <= g.Clone; i++ {
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

	var quietDays int
	for day := 0; len(g.participants) > g.VictorCount; day++ {
		time.Sleep(g.DayDelay)

		select {
		case <-ctx.Done():
			g.logMessage(log.InfoLevel, "context done, cancelling game on day %v", day)
			g.Lock()
			g.state = Cancelled
			g.Unlock()
			return nil

		default:
			g.logMessage(log.InfoLevel, "simulating day %v with %v tributes", day, len(g.participants))
			var err error

			var mustKill bool
			if quietDays >= settings.MaxQuietDays {
				mustKill = true
			}

			pcount := len(g.participants)
			g.participants, err = g.runDay(ctx, day, g.participants, mustKill)
			if err != nil {
				g.logMessage(log.ErrorLevel, "failed to simulate day %v: %v", day, err)
				g.Sender.SendQuoted(fmt.Sprintf("failed to run game for day %v", day+1))
				g.Lock()
				g.state = Cancelled
				g.Unlock()
				return nil
			}

			if len(g.participants) == pcount {
				quietDays++
			} else {
				quietDays = 0
			}

			g.logMessage(log.InfoLevel, "users left after day %v: %v", day, len(g.participants))
		}
	}

	var mentions []string
	var winnerLogs []string
	for _, p := range g.participants {
		mentions = append(mentions, p.Mention())
		winnerLogs = append(winnerLogs, p.DisplayFullName())
	}

	g.logMessage(log.InfoLevel, fmt.Sprintf("Winners for sponsor %v: %v", g.Sponsor, strings.Join(winnerLogs, ",")))

	mentionStr := strings.Join(mentions, ", ")
	snow := settings.GetEmoji(settings.EmojiPresSnow)
	host := settings.GetEmoji(settings.EmojiCaesar)
	lines := []string{
		fmt.Sprintf("%v  This year's Hunger Games have concluded. Congratulations to our new victor(s): %v!", host.EmojiCode(), mentionStr),
		settings.WhiteSpaceChar,
		fmt.Sprintf("%v  The tributes demonstrated exceptional survival skills and the winner(s) emerged victorious. Their combat prowess is a testament to the superiority of the Capitol's training and preparation methods.", snow.EmojiCode()),
		settings.WhiteSpaceChar,
		fmt.Sprintf("The victor(s) have won **%s**!", g.Sponsor),
	}

	if g.Notify != nil {
		lines = append(
			lines,
			settings.WhiteSpaceChar,
			fmt.Sprintf("(fyi <@%v>)", g.Notify.ID),
		)
	}

	g.sendBatchOutput(lines)

	g.Lock()
	g.state = Finished
	g.Unlock()

	return g.participants
}

func (g *Game) runDay(ctx context.Context, day int, participants []*Participant, mustKill bool) ([]*Participant, error) {
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
	if mustKill {
		g.logMessage(log.InfoLevel, "forcing a kill due to too many quiet days")
		min = 1
	}

	max := int(math.Min(
		float64(len(participants)/2),
		float64(len(participants)-g.VictorCount),
	))
	max++ // +1 b/c right is exclusive

	// 1/2 - 3/4 on the first day, it's always a slaughter!
	if day == 0 && len(participants) > 5 {
		min = max / 2
		max = int(math.Min(
			float64(max*3/4),
			float64(len(participants)-g.VictorCount),
		))
		max++ // +1 b/c right is exclusive
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

	var deadNames []string
	for i := range dead {
		deadNames = append(deadNames, participants[i].DisplayName())

		mention := ""
		if g.Clone == 1 {
			mention = participants[i].Mention()
		}

		line := "• " + g.PhraseGenerator.GetRandomPhrase(participants[i].DisplayName(), mention, livingNames)
		g.logMessage(log.TraceLevel, "Day %v: %v", day, line)
		output = append(output, line)
	}

	g.logMessage(log.DebugLevel, "Dead players after day %v: %v", day+1, strings.Join(deadNames, ", "))

	host := settings.GetEmoji(settings.EmojiCaesar)
	output = append(output, settings.WhiteSpaceChar, fmt.Sprintf(
		"%v  %v player(s) remain at the end of day %v: %v",
		host.EmojiCode(),
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
	hostEmoji := settings.GetEmoji(settings.EmojiCaesar)
	g.logMessage(log.DebugLevel, "tribute count: %v", len(participants))

	tributeLines := []string{
		fmt.Sprintf("%v  Please welcome our brave tributes!", hostEmoji.EmojiCode()),
		settings.WhiteSpaceChar,
		"What a fantastic group of individuals we have for this year's contest:",
	}

	var tributes []string
	for _, p := range participants {
		tributes = append(tributes, p.DisplayName())
	}

	tributeLines = append(tributeLines, strings.Join(tributes, ", "))

	if g.Clone > 1 {
		cloneEmoji := settings.GetEmoji(settings.EmojiClone)
		cloneCode := cloneEmoji.EmojiCode()
		tributeLines = append(tributeLines, settings.WhiteSpaceChar, fmt.Sprintf(
			"%s   **MEGA HG MODE ACTIVATED -- TRIBUTES WILL BE CLONED %v TIMES**   %v",
			cloneCode, g.Clone, cloneCode))
	}

	g.sendBatchOutput(tributeLines)
}

func (g *Game) sendBatchOutput(lines []string) {
	g.Sender.SendQuoted(strings.Join(lines, "\n"))
}

func (g *Game) logMessage(level log.Level, msg string, args ...interface{}) {
	// msg = fmt.Sprintf("%v (server:'%s' channel:'%s')", msg, g.serverName, g.channelName)
	switch level {
	case log.TraceLevel:
		log.Tracef(msg, args...)
	case log.DebugLevel:
		log.Debugf(msg, args...)
	case log.InfoLevel:
		log.Infof(msg, args...)
	case log.WarnLevel:
		log.Warnf(msg, args...)
	case log.ErrorLevel:
		log.Errorf(msg, args...)
	case log.PanicLevel:
		log.Panicf(msg, args...)
	}
}
