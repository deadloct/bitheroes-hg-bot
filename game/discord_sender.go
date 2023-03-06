package game

import (
	"github.com/bwmarrin/discordgo"
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
	close(s.stopCh)
}

func (s *DiscordSender) listen() {
	for {
		select {
		case str := <-s.sendCh:
			s.session.ChannelMessageSend(s.channelID, str)
		case <-s.stopCh:
			return
		}
	}
}

func (s *DiscordSender) Send(str string) (*discordgo.Message, error) {
	log.Debugf("sending message of length %v", len(str))
	msg, err := s.session.ChannelMessageSend(s.channelID, str)
	if err != nil {
		log.Errorf("error sending message of length %v: %v", len(str), err)
	} else {
		log.Debugf("successfully sent message of length %v", len(str))
	}

	return msg, err
}
