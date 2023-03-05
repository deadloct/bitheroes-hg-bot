package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type TemplateValues struct {
	User  string
	Delay time.Duration
}

type Game struct {
	Settings *Settings
}

func NewGame(s *Settings) *Game {
	return &Game{Settings: s}
}

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
	intro, err := g.getIntro(TemplateValues{
		User:  mc.Author.Username,
		Delay: delay,
	})
	if err != nil {
		return nil, err
	}

	msg, err := session.ChannelMessageSend(mc.ChannelID, intro)
	if err != nil {
		return nil, err
	}

	return g.delayedStart(delay, session, msg), nil
}

func (g *Game) getIntro(vals TemplateValues) (string, error) {
	introPath := path.Join(SettingsLocation, g.Settings.IntroTemplate)
	v, err := ioutil.ReadFile(introPath)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %w", introPath, err)
	}

	tmplStr := string(v[:])
	tmpl, err := template.New(g.Settings.IntroTemplate).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, vals)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func (g *Game) delayedStart(delay time.Duration, session *discordgo.Session, msg *discordgo.Message) chan struct{} {
	stop := make(chan struct{})

	go func() {
		select {
		case <-stop:
			log.Info("game cancelled")
		case <-time.After(delay):
			g.startGame(session, msg, stop)
		}
	}()

	return stop
}

func (g *Game) startGame(session *discordgo.Session, msg *discordgo.Message, stop chan struct{}) {
	users, err := session.MessageReactions(msg.ChannelID, msg.ID, ParticipantEmoji, 100, "", "")
	if err != nil {
		log.Fatalf("could not retrieve reactions: %v", err)
	}

	// game logic here

	var mentions []string
	for _, u := range users {
		mentions = append(mentions, fmt.Sprintf("<@%v>", u.ID))
	}

	session.ChannelMessageSend(msg.ChannelID, "Congrats to the winner(s): "+strings.Join(mentions, ", "))
}
