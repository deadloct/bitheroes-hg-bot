package game

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
	"github.com/deadloct/discord-squid-game/data"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultStartDelay = 20 * time.Second
	MinimumStartDelay = 30  // Seconds
	MaximumStartDelay = 600 // 10 minutes in seconds
)

var DataLocation = path.Join(".", "data")

type TemplateValues struct {
	User string
}

type Game struct {
	Mode *data.Mode
}

func NewGame(m *data.Mode) *Game {
	return &Game{Mode: m}
}

func (g *Game) Start(session *discordgo.Session, mc *discordgo.MessageCreate) (chan struct{}, error) {
	vals := TemplateValues{
		User: mc.Author.Username,
	}

	// This is the welcome messsage that people react to to enter.
	intro, err := g.getIntro(vals)
	if err != nil {
		return nil, err
	}

	msg, err := session.ChannelMessageSend(mc.ChannelID, intro)
	if err != nil {
		return nil, err
	}

	// The start delay gives people time to react before the game starts
	delay := DefaultStartDelay
	parts := strings.Split(mc.Content, " ")
	if len(parts) > 1 {
		if num, err := strconv.Atoi(parts[1]); err == nil && num >= MinimumStartDelay && num <= MaximumStartDelay {
			delay = time.Duration(num) * time.Second
		}
	}

	return g.delayedStart(delay, session, msg), nil
}

func (g *Game) getIntro(vals TemplateValues) (string, error) {
	introPath := path.Join(DataLocation, g.Mode.IntroTemplate)
	v, err := ioutil.ReadFile(introPath)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %w", introPath, err)
	}

	tmplStr := string(v[:])
	tmpl, err := template.New(g.Mode.IntroTemplate).Parse(tmplStr)
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
	users, err := session.MessageReactions(msg.ChannelID, msg.ID, "🦑", 100, "", "")
	if err != nil {
		log.Fatalf("could not retrieve reactions: %v", err)
	}

	var userStrings []string
	for _, u := range users {
		userStrings = append(userStrings, fmt.Sprintf("%v#%v", u.Username, u.ID))
	}

	_, err = session.ChannelMessageSend(msg.ChannelID, "Contestants: "+strings.Join(userStrings, ", "))
	if err != nil {
		log.Error(err)
	}
}
