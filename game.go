package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Game struct{}

func NewGame() *Game { return &Game{} }

func (g *Game) Start(session *discordgo.Session, mc *discordgo.MessageCreate) (chan struct{}, error) {
	// The start delay gives people time to react before the game starts
	delay := DefaultStartDelay
	parts := strings.Split(mc.Content, " ")
	if len(parts) > 1 {
		if num, err := strconv.Atoi(parts[1]); err == nil && num >= MinimumStartDelay && num <= MaximumStartDelay {
			delay = time.Duration(num) * time.Second
		}
	}

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(IntroValues{
		User:  mc.Author.Username,
		Delay: delay,
	})
	if err != nil {
		return nil, err
	}

	log.Debugf("sending intro to channel %v: %v", mc.ChannelID, intro)
	msg, err := session.ChannelMessageSend(mc.ChannelID, intro)
	if err != nil {
		return nil, err
	}

	return g.delayedStart(delay, session, msg), nil
}

func (g *Game) getIntro(vals IntroValues) (string, error) {
	var result bytes.Buffer
	if err := Intro.Execute(&result, vals); err != nil {
		return "", err
	}

	return result.String(), nil
}

func (g *Game) delayedStart(delay time.Duration, session *discordgo.Session, msg *discordgo.Message) chan struct{} {
	log.Debugf("delaying start by %v", delay)

	stop := make(chan struct{})
	go func() {
		select {
		case <-stop:
			log.Info("game cancelled")
		case <-time.After(delay):
			g.launchGame(session, msg, stop)
		}
	}()

	return stop
}

func (g *Game) launchGame(session *discordgo.Session, msg *discordgo.Message, stop chan struct{}) error {
	log.Debug("starting game")

	users, err := session.MessageReactions(msg.ChannelID, msg.ID, ParticipantEmoji, 100, "", "")
	if err != nil {
		log.Fatalf("could not retrieve reactions: %v", err)
	}

	if len(users) == 0 {
		session.ChannelMessageSend(msg.ChannelID, "No users entered the contest")
		return nil
	}

	// TODO: Remove this
	for i := 1; i < 10; i++ {
		users = append(users, &discordgo.User{
			Username: fmt.Sprintf("%v-%v", users[0].Username, i),
			ID:       users[0].ID,
		})
	}

	log.Debugf("tribute count: %v", len(users))

	out := make(chan string)
	go g.handleOutput(session, msg, out, stop)

	for day := 0; len(users) > 1; day++ {
		select {
		case <-stop:
			log.Info("game cancelled")
		default:
			log.Debugf("simulating day %v with %v tributes", day, len(users))
			users, err = g.simulateDay(day, users, out)
			if err != nil {
				log.Errorf("failed to simulate day %v: %v", day, err)
			}

			log.Debugf("users left after day %v: %v", day, len(users))
			time.Sleep(DayDelay)
		}
	}

	var mentions []string
	for _, u := range users {
		mentions = append(mentions, fmt.Sprintf("<@%v>", u.ID))
	}

	log.Debugf("winners: %v", mentions)
	out <- "---\nThis year's Hunger Games have concluded.\nCongratulations to our new victor(s): " + strings.Join(mentions, ", ")
	close(stop)
	return nil
}

func (g *Game) handleOutput(session *discordgo.Session, msg *discordgo.Message, out chan string, stop chan struct{}) {
	for {
		select {
		case str := <-out:
			log.Debugf("sending message '%v'...", str)
			if _, err := session.ChannelMessageSend(msg.ChannelID, str); err != nil {
				log.Errorf("error sending message '%v': %v", str, err)
			} else {
				log.Debugf("message sent '%v'", str)
			}
		case <-stop:
			return
		}
	}
}

func (g *Game) simulateDay(day int, users []*discordgo.User, outCh chan string) ([]*discordgo.User, error) {
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
		g.send(output, outCh)
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

	g.send(output, outCh)
	return living, nil
}

func (g *Game) getRandomPhrase(dying *discordgo.User, alive []*discordgo.User) string {
	defaultPhrase := fmt.Sprintf("%v died of dysentery.", dying.Username)

	killer := "another player"
	if killerNum, err := GetRandomInt(0, len(alive)); err == nil {
		killer = alive[killerNum].Username
	}

	tmplNum, err := GetRandomInt(0, len(Phrases))
	if err != nil {
		log.Errorf("could not retrieve random int for picking a phrase: %v", err)
		return defaultPhrase
	}

	vals := PhraseValues{
		Killer: killer,
		Dying:  dying.Username,
	}
	var result bytes.Buffer
	if err := Phrases[tmplNum].Execute(&result, vals); err != nil {
		log.Errorf("error executing template with vals: %v", err)
		return defaultPhrase
	}

	return result.String()
}

func (g *Game) send(output []string, out chan string) {
	lines := []string{"---"}
	for i := 0; i < len(output); i++ {
		lines = append(lines, "> "+output[i])
	}

	out <- strings.Join(lines, "\n")
}
