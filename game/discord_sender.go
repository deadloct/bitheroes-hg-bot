package game

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

type Sender interface {
	Start() chan string
	Stop()
	Send(str string) (*discordgo.Message, error)
}

type DiscordSender struct {
	session   *discordgo.Session
	channelID string
	sendCh    chan string
	stopCh    chan struct{}
}

func NewDiscordSender(session *discordgo.Session, channelID string) *DiscordSender {
	return &DiscordSender{
		channelID: channelID,
		session:   session,
	}
}

func (s *DiscordSender) Start() chan string {
	s.sendCh = make(chan string)
	s.stopCh = make(chan struct{})
	go s.listen()
	return s.sendCh
}

func (s *DiscordSender) Stop() {
	close(s.sendCh)
	close(s.stopCh)
}

func (s *DiscordSender) listen() {
	for {
		select {
		case str := <-s.sendCh:
			if str != "" {
				s.Send(str)
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *DiscordSender) Send(str string) (*discordgo.Message, error) {
	if len(str) <= settings.DiscordMaxMessageLength {
		return s.send(str)
	}

	var lineStart string
	if strings.HasPrefix(str, "> ") {
		lineStart = "> "
	}

	words := strings.Fields(str)
	var (
		line string
		msg  *discordgo.Message
		err  error
	)

	for _, word := range words {
		separator := " "

		// Space character between line and word is why this uses >= instead of >
		if len(line)+len(word) >= settings.DiscordMaxMessageLength {
			if msg, err = s.send(line); err != nil {
				return nil, err
			}

			line = ""
			separator = lineStart
		}

		line = fmt.Sprintf("%s%s%s", line, separator, word)
	}

	if len(line) > 0 {
		if msg, err = s.send(line); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (s *DiscordSender) send(str string) (*discordgo.Message, error) {
	log.Tracef("sending message of length %v", len(str))
	msg, err := s.session.ChannelMessageSend(s.channelID, str)
	if err != nil {
		log.Errorf("error sending message of length %v: %v", len(str), err)
	} else {
		log.Tracef("successfully sent message of length %v", len(str))
	}

	return msg, err
}
