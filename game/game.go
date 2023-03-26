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
	Author          *discordgo.User
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

	channelName  string
	introMessage *discordgo.Message
	sendCh       chan string
	serverName   string
	state        GameState
	users        []*discordgo.User
	userMap      map[string]*discordgo.User

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
		GameConfig:  cfg,
		userMap:     make(map[string]*discordgo.User),
		channelName: channelName,
		serverName:  serverName,
	}
}

func (g *Game) Start(ctx context.Context) error {

	g.sendCh = g.Sender.Start()

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(settings.IntroValues{
		Delay:       g.Delay,
		EmojiCode:   settings.ParticipantEmojiCode,
		User:        g.Author.Username,
		VictorCount: g.VictorCount,
	})
	if err != nil {
		return err
	}

	g.logMessage(log.DebugLevel, "sending intro")
	if g.introMessage, err = g.Sender.Send(intro); err != nil {
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

func (g *Game) RegisterUser(messageID, emoji string, user *discordgo.User) {
	if g.HasStarted() {
		return
	}

	if emoji != settings.ParticipantEmojiName {
		return
	}

	if user.Bot {
		return
	}

	if _, ok := g.userMap[user.ID]; !ok {
		g.userMap[user.ID] = user
		g.users = append(g.users, user)
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
	g.startRandomJokes(ctx, jokeCh)

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

func (g *Game) startRandomJokes(ctx context.Context, stop chan struct{}) {
	if g.JokeGenerator == nil {
		g.logMessage(log.InfoLevel, "not starting joke engine because joke generator is nil for game in channel %s", g.introMessage.ChannelID)
		return
	}

	go func() {
		ticker := time.NewTicker(settings.JokeInterval)
		time.Sleep(2 * time.Second)
		msg, err := g.sendJoke(nil)
		if err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				if msg, err = g.sendJoke(msg); err != nil {
					return
				}
			}
		}
	}()
}

func (g *Game) sendJoke(msg *discordgo.Message) (*discordgo.Message, error) {
	prefix := "Greetings, esteemed guests and citizens of the Capitol! As the royal jester to our illustrious President Snow, I stand before you today to bring some much-needed levity and humor to this esteemed gathering. I understand that some of you may be feeling impatient, but fear not! I am here to entertain you with the finest collection of dad jokes this side of the Districts."

	j, err := g.JokeGenerator.GetJoke()
	if err != nil {
		g.logMessage(log.WarnLevel, "failed to get joke: %v", err)
		return msg, err
	}

	text := fmt.Sprintf("%v\n\n%v\n\n%v\n\n%v\n\n%v\n\n%v", settings.DefaultSeparator, prefix, settings.HalfSeparator, j.Question, j.Answer, settings.DefaultSeparator)
	g.logMessage(log.DebugLevel, "Sending joke '%v'", j)
	if msg == nil {
		if msg, err = g.Sender.Send(text); err != nil {
			g.logMessage(log.WarnLevel, "failed to send joke: %v", err)
			return msg, err
		}
	} else {
		if msg, err = g.Session.ChannelMessageEdit(msg.ChannelID, msg.ID, text); err != nil {
			g.logMessage(log.WarnLevel, "failed to edit joke: %v", err)
			return msg, err
		}
	}

	return msg, nil
}

func (g *Game) run(ctx context.Context) {
	g.logMessage(log.InfoLevel, "starting game with user count %v", len(g.users))
	g.Lock()
	g.state = Started
	g.Unlock()

	if len(g.users) == 0 {
		g.logMessage(log.InfoLevel, "no users entered")
		g.Sender.Send(fmt.Sprintf("No tributes have come forward within %v. This district will be eliminated.", g.Delay))
		g.Lock()
		g.state = Cancelled
		g.Unlock()
		return
	}

	if g.EntryMultiplier > 1 {
		for _, u := range g.users {
			for i := 2; i <= g.EntryMultiplier; i++ {
				g.users = append(g.users, &discordgo.User{
					Username:      fmt.Sprintf("%v-%v", u.Username, i),
					ID:            u.ID,
					Discriminator: u.Discriminator,
				})
			}
		}
	}

	g.sendTributeOutput(g.users)

	for day := 0; len(g.users) > g.VictorCount; day++ {
		time.Sleep(g.DayDelay)

		select {
		case <-ctx.Done():
			g.logMessage(log.InfoLevel, "context done, cancelling game on day %v", day)
			g.Lock()
			g.state = Cancelled
			g.Unlock()
			return

		default:
			g.logMessage(log.InfoLevel, "simulating day %v with %v tributes", day, len(g.users))
			var err error
			g.users, err = g.runDay(ctx, day, g.users)
			if err != nil {
				g.logMessage(log.ErrorLevel, "failed to simulate day %v: %v", day, err)
				g.Sender.Send(fmt.Sprintf("failed to run game for day %v", day+1))
				g.Lock()
				g.state = Cancelled
				g.Unlock()
				return
			}

			g.logMessage(log.InfoLevel, "users left after day %v: %v", day, len(g.users))
		}
	}

	var mentions []string
	for _, u := range g.users {
		mentions = append(mentions, fmt.Sprintf("<@%v>", u.ID))
		g.logMessage(log.InfoLevel, "winner: %v#%v (%v)", u.Username, u.Discriminator, u.ID)
	}

	congrats := lib.ToDoubleStruck("Congratulations to our new victor(s)")
	g.sendBatchOutput([]string{
		settings.DefaultSeparator,
		"This year's Hunger Games have concluded.",
		fmt.Sprintf("**%v**: %v", congrats, strings.Join(mentions, ", ")),
		settings.DefaultSeparator,
	})

	g.Lock()
	g.state = Finished
	g.Unlock()
}

func (g *Game) runDay(ctx context.Context, day int, users []*discordgo.User) ([]*discordgo.User, error) {
	if len(users) == 1 {
		return users, nil
	}

	output := []string{
		settings.HalfSeparator,
		lib.ToDoubleStruck(fmt.Sprintf("**Day %v**", day+1)),
		settings.HalfSeparator,
		" ",
	}

	// min and max are 0-based
	var min int
	max := len(users) - g.VictorCount + 1 // right is exclusive, so this is really `len(users) - g.VictorCount``

	// 1/2 - 3/4 on the first day, it's always a slaughter
	if day == 0 && len(users) > 5 {
		min = max / 2
		max = max * 3 / 4
	}

	killCount, err := lib.GetRandomInt(min, max)
	if err != nil {
		g.logMessage(log.ErrorLevel, "failed to get random number between %v and %v: %v", min, max, err)
		return nil, err
	}

	if killCount == 0 {
		output = append(output, fmt.Sprintf("**All was quiet on day %v.**", day+1))
		g.sendBatchOutput(output)
		return users, nil
	}

	dead := make(map[int]struct{})
	for i := 0; i < killCount; i++ {
		for {
			toDie, err := lib.GetRandomInt(0, len(users))
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

	var living []*discordgo.User
	var livingNames []string
	for i, user := range users {
		if _, dead := dead[i]; !dead {
			living = append(living, user)
			livingNames = append(livingNames, user.Username)
		}
	}

	for i := range dead {
		mention := ""
		if g.EntryMultiplier == 1 {
			mention = fmt.Sprintf("<@%v>", users[i].ID)
		}

		line := "• " + g.PhraseGenerator.GetRandomPhrase(users[i].Username, mention, livingNames)
		g.logMessage(log.DebugLevel, "Day %v: %v", day, line)
		output = append(output, line)
	}

	output = append(
		output,
		settings.DefaultSeparator,
		fmt.Sprintf("**%v player(s) remain at the end of day %v:** %v", len(living), day+1, strings.Join(livingNames, ", ")),
	)

	select {
	case <-ctx.Done():
	default:
		g.sendBatchOutput(output)
	}

	return living, nil
}

func (g *Game) sendTributeOutput(users []*discordgo.User) {
	g.logMessage(log.DebugLevel, "tribute count: %v", len(users))

	tributeLines := []string{
		settings.DefaultSeparator,
		lib.ToDoubleStruck("**Please welcome our brave tributes!**"),
		settings.DefaultSeparator,
		"",
		"What a fantastic group of individuals we have for this year's contest:",
	}
	var tributes []string
	for _, u := range users {
		tributes = append(tributes, u.Username)
	}

	tributeLines = append(tributeLines, strings.Join(tributes, ", "), "", settings.DefaultSeparator)
	g.sendBatchOutput(tributeLines)
}

func (g *Game) sendBatchOutput(lines []string) {
	var output string

	for _, line := range lines {
		next := fmt.Sprintf("> %v", line)

		if len(output)+len(next) > settings.DiscordMaxMessageLength {
			g.sendCh <- output
			output = ""
		}

		output += next + "\n"
	}

	if len(output) > 0 {
		g.sendCh <- output
	}
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
