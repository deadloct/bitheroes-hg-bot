package game

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deadloct/bitheroes-hg-bot/settings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Game struct {
	author *discordgo.User
	cmd    string
	sender Sender

	introMessage *discordgo.Message
	sendCh       chan string
	session      *discordgo.Session
}

func NewGame(session *discordgo.Session, author *discordgo.User, cmd string, sender Sender) *Game {
	return &Game{author: author, cmd: cmd, sender: sender, session: session}
}

func (g *Game) Start() (chan struct{}, error) {
	g.sendCh = g.sender.Start()

	// The start delay gives people time to react before the game starts
	delay := settings.DefaultStartDelay
	parts := strings.Split(g.cmd, " ")
	if len(parts) > 1 {
		if num, err := strconv.Atoi(parts[1]); err == nil && num >= settings.MinimumStartDelay && num <= settings.MaximumStartDelay {
			delay = time.Duration(num) * time.Second
		}
	}

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(settings.IntroValues{
		User:  g.author.Username,
		Delay: delay,
	})
	if err != nil {
		return nil, err
	}

	log.Debug("sending intro")
	if g.introMessage, err = g.sender.Send(intro); err != nil {
		return nil, err
	}

	return g.delayedStart(delay), nil
}

func (g *Game) getIntro(vals settings.IntroValues) (string, error) {
	var result bytes.Buffer
	if err := settings.Intro.Execute(&result, vals); err != nil {
		return "", err
	}

	return result.String(), nil
}

func (g *Game) delayedStart(delay time.Duration) chan struct{} {
	log.Debugf("delaying start by %v", delay)

	stop := make(chan struct{})
	go func() {
		select {
		case <-stop:
			log.Info("game cancelled")
		case <-time.After(delay):
			g.run(stop)
		}
	}()

	return stop
}

func (g *Game) run(stop chan struct{}) error {
	log.Debug("starting game")

	// TODO: only retrieves 100, need to get more somehow
	// TODO: don't directly access session to remove dependency and make testing easier
	users, err := g.session.MessageReactions(g.introMessage.ChannelID, g.introMessage.ID,
		settings.ParticipantEmoji, 100, "", "")
	if err != nil {
		log.Fatalf("could not retrieve reactions: %v", err)
	}

	if len(users) == 0 {
		g.sender.Send("No users entered the contest")
		return nil
	}

	// TODO: Remove this testing code
	for i := 1; i < 120; i++ {
		users = append(users, &discordgo.User{
			Username: fmt.Sprintf("%v-%v", users[0].Username, i),
			ID:       users[0].ID,
		})
	}

	log.Debugf("tribute count: %v", len(users))

	for day := 0; len(users) > 1; day++ {
		if day > 0 {
			time.Sleep(settings.DayDelay)
		}

		select {
		case <-stop:
			log.Info("game cancelled")
		default:
			log.Debugf("simulating day %v with %v tributes", day, len(users))
			users, err = g.runDay(day, users)
			if err != nil {
				log.Errorf("failed to simulate day %v: %v", day, err)
			}

			log.Debugf("users left after day %v: %v", day, len(users))
		}
	}

	var mentions []string
	for _, u := range users {
		mentions = append(mentions, fmt.Sprintf("<@%v>", u.ID))
	}

	log.Debugf("winners: %v", mentions)
	g.sendCh <- `---
> This year's Hunger Games have concluded.
> 
> Congratulations to our new victor(s): ` + strings.Join(mentions, ", ")
	close(stop)
	return nil
}

func (g *Game) runDay(day int, users []*discordgo.User) ([]*discordgo.User, error) {
	if len(users) == 1 {
		return users, nil
	}

	output := []string{fmt.Sprintf("Beginning day %v...", day+1)}

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
		output = append(output, fmt.Sprintf("All was quiet on day %v", day+1))
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
		fmt.Sprintf("Players remaining at the end of day %v: %v", day+1, strings.Join(livingNames, ", ")),
	)

	g.sendDayOutput(output)
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
		Dying:  dying.Username,
	}
	var result bytes.Buffer
	if err := settings.Phrases[tmplNum].Execute(&result, vals); err != nil {
		log.Errorf("error executing template with vals: %v", err)
		return defaultPhrase
	}

	return result.String()
}

func (g *Game) sendDayOutput(lines []string) {
	output := "---"

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
