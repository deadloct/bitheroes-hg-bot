package game

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

type GameConfig struct {
	Author  *discordgo.User
	Delay   time.Duration // delayed start
	Sender  Sender
	Session *discordgo.Session
}

type Game struct {
	GameConfig

	introMessage *discordgo.Message
	sendCh       chan string
	state        GameState

	sync.Mutex
}

func NewGame(cfg GameConfig) *Game {
	return &Game{GameConfig: cfg}
}

func (g *Game) Start(ctx context.Context) error {
	g.sendCh = g.Sender.Start()

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(settings.IntroValues{
		User:  g.Author.Username,
		Delay: g.Delay,
	})
	if err != nil {
		return err
	}

	log.Debug("sending intro")
	if g.introMessage, err = g.Sender.Send(intro); err != nil {
		return err
	}

	g.Session.MessageReactionAdd(g.introMessage.ChannelID, g.introMessage.ID, settings.ParticipantEmoji)

	g.delayedStart(ctx)
	return nil
}

func (g *Game) GetState() GameState {
	g.Lock()
	defer g.Unlock()

	return g.state
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

func (g *Game) HasEnded() bool {
	g.Lock()
	defer g.Unlock()

	return g.state == Finished || g.state == Cancelled
}

func (g *Game) getIntro(vals settings.IntroValues) (string, error) {
	var result bytes.Buffer
	if err := settings.Intro.Execute(&result, vals); err != nil {
		return "", err
	}

	return result.String(), nil
}

func (g *Game) delayedStart(ctx context.Context) {
	log.Debugf("delaying start by %v", g.Delay)

	go func() {
		select {
		case <-ctx.Done():
			log.Info("context done, cancelling game")
			g.Lock()
			g.state = Cancelled
			g.Unlock()

		case <-time.After(g.Delay):
			g.run(ctx)
		}
	}()
}

func (g *Game) run(ctx context.Context) {
	log.Debug("starting game")
	g.Lock()
	g.state = Started
	g.Unlock()

	// TODO: only retrieves 100, need to get more somehow
	// TODO: don't directly access session to remove dependency and make testing easier
	usersWithBot, err := g.Session.MessageReactions(g.introMessage.ChannelID, g.introMessage.ID,
		settings.ParticipantEmoji, 100, "", "")
	if err != nil {
		log.Fatalf("could not retrieve reactions: %v", err)
	}

	var users []*discordgo.User
	for _, u := range usersWithBot {
		if !u.Bot {
			users = append(users, u)
		}
	}

	if len(users) == 0 {
		g.Sender.Send(fmt.Sprintf("No tributes have come forward within %v. This district will be eliminated.", g.Delay))
		g.Lock()
		g.state = Cancelled
		g.Unlock()
		return
	}

	// TODO: Remove this testing code
	for _, u := range users {
		for i := 2; i < 6; i++ {
			users = append(users, &discordgo.User{
				Username: fmt.Sprintf("%v-%v", u.Username, i),
				ID:       u.ID,
			})
		}
	}

	g.sendTributeOutput(users)

	for day := 0; len(users) > 1; day++ {
		time.Sleep(settings.DayDelay)

		select {
		case <-ctx.Done():
			log.Infof("context done, cancelling game on day %v", day)
			g.Lock()
			g.state = Cancelled
			g.Unlock()
			return

		default:
			log.Debugf("simulating day %v with %v tributes", day, len(users))
			users, err = g.runDay(ctx, day, users)
			if err != nil {
				log.Errorf("failed to simulate day %v: %v", day, err)
				g.Sender.Send(fmt.Sprintf("failed to run game for day %v", day+1))
				g.Lock()
				g.state = Cancelled
				g.Unlock()
				return
			}

			log.Debugf("users left after day %v: %v", day, len(users))
		}
	}

	var mentions []string
	for _, u := range users {
		mentions = append(mentions, fmt.Sprintf("<@%v>", u.ID))
	}

	log.Debugf("winners: %v", mentions)
	g.sendCh <- strings.Join([]string{
		"> " + settings.DefaultSeparator,
		"> This year's Hunger Games have concluded.",
		"> ",
		fmt.Sprintf("> **Congratulations to our new victor(s), %v!**", strings.Join(mentions, ", ")),
		"> " + settings.DefaultSeparator,
	}, "\n")

	g.Lock()
	g.state = Finished
	g.Unlock()
}

func (g *Game) runDay(ctx context.Context, day int, users []*discordgo.User) ([]*discordgo.User, error) {
	if len(users) == 1 {
		return users, nil
	}

	output := []string{
		settings.DefaultSeparator,
		fmt.Sprintf("**𝔹𝕖𝕘𝕚𝕟𝕟𝕚𝕟𝕘 𝔻𝕒𝕪 %v**", day+1),
		settings.DefaultSeparator,
		" ",
	}

	// min and max are 0-based
	var min int
	max := len(users) // right is exclusive, so this is really `len(users) - 1``

	// 1/2 - 3/4 on the first day, it's always a slaughter
	if day == 0 {
		min = len(users) / 2
		max = len(users) * 3 / 4
	}

	killCount, err := GetRandomInt(min, max)
	if err != nil {
		log.Errorf("failed to get random number between %v and %v: %v", min, max, err)
		return nil, err
	}

	if killCount == 0 {
		output = append(output, fmt.Sprintf("**All was quiet on day %v.**", day+1))
		g.sendDayOutput(output)
		return users, nil
	}

	dead := make(map[int]struct{})
	for i := 0; i < killCount; i++ {
		for {
			toDie, err := GetRandomInt(0, len(users))
			if err != nil {
				log.Errorf("failed to get random number between %v and %v: %v", min, max, err)
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
		output = append(output, g.getRandomPhrase(users[i], living))
	}

	output = append(
		output,
		"",
		fmt.Sprintf("**Players remaining at the end of day %v:** %v", day+1, strings.Join(livingNames, ", ")),
	)

	select {
	case <-ctx.Done():
	default:
		g.sendDayOutput(output)
	}

	return living, nil
}

func (g *Game) getRandomPhrase(dying *discordgo.User, alive []*discordgo.User) string {
	defaultPhrase := fmt.Sprintf("%v died of dysentery.", dying.Username)

	killer := "another player"
	if killerNum, err := GetRandomInt(0, len(alive)); err == nil {
		killer = alive[killerNum].Username
	}

	tmplNum, err := GetRandomInt(0, len(settings.Phrases))
	if err != nil {
		log.Errorf("could not retrieve random int for picking a phrase: %v", err)
		return defaultPhrase
	}

	vals := settings.PhraseValues{
		Killer: killer,
		Dying:  fmt.Sprintf("<@%v>", dying.ID),
	}
	var result bytes.Buffer
	if err := settings.Phrases[tmplNum].Execute(&result, vals); err != nil {
		log.Errorf("error executing template with vals: %v", err)
		return defaultPhrase
	}

	return result.String()
}

func (g *Game) sendTributeOutput(users []*discordgo.User) {
	log.Debugf("tribute count: %v", len(users))

	tributeLines := []string{
		settings.DefaultSeparator,
		"Please welcome our brave tributes!",
		settings.DefaultSeparator,
		"",
		"What a fantastic group of individuals we have for this year's contest:",
	}
	var tributes []string
	for _, u := range users {
		tributes = append(tributes, u.Username)
	}

	tributeLines = append(tributeLines, strings.Join(tributes, ", "), "", settings.DefaultSeparator)
	g.sendDayOutput(tributeLines)
}

func (g *Game) sendDayOutput(lines []string) {
	var output string

	for _, line := range lines {
		next := fmt.Sprintf("\n> %v", line)

		if len(output)+len(next) > settings.DiscordMaxMessageLength {
			g.sendCh <- output
			output = ""
		}

		output += next
	}

	if len(output) > 0 {
		g.sendCh <- output
	}
}
