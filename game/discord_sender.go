package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

type Sender interface {
	SendNormal(str string) (*discordgo.Message, error)
	SendEmbed(str string) (*discordgo.Message, error)
}

type DiscordSender struct {
	channelID string
	session   *discordgo.Session
}

func NewDiscordSender(session *discordgo.Session, channelID string) *DiscordSender {
	return &DiscordSender{
		channelID: channelID,
		session:   session,
	}
}

func (s *DiscordSender) SendNormal(str string) (*discordgo.Message, error) {
	return s.splitSend(str, s.normal)
}

func (s *DiscordSender) SendEmbed(str string) (*discordgo.Message, error) {
	return s.splitSend(str, s.embed)
}

func (s *DiscordSender) splitSend(str string, send func(str string) (*discordgo.Message, error)) (*discordgo.Message, error) {
	if len(str) <= settings.DiscordMaxMessageLength {
		return send(str)
	}

	words := strings.Fields(str)
	var (
		line string
		msg  *discordgo.Message
		err  error
	)

	for _, word := range words {
		// Space character between line and word is why this uses >= instead of >
		if len(line)+len(word) >= settings.DiscordMaxMessageLength {
			if msg, err = send(line); err != nil {
				return nil, err
			}

			line = ""
		}

		line = fmt.Sprintf("%s%s%s", line, " ", word)
	}

	if len(line) > 0 {
		if msg, err = send(line); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (s *DiscordSender) normal(str string) (*discordgo.Message, error) {
	quoted := s.addBQ(str)
	log.Tracef("sending message of length %v", len(quoted))
	msg, err := s.session.ChannelMessageSend(s.channelID, quoted)
	if err != nil {
		log.Errorf("error sending message of length %v: %v", len(quoted), err)
	} else {
		log.Tracef("successfully sent message of length %v", len(quoted))
	}

	err = errors.Join(err, s.sendBlankLine())
	return msg, err
}

func (s *DiscordSender) embed(str string) (*discordgo.Message, error) {
	log.Tracef("sending message of length %v", len(str))
	msg, err := s.session.ChannelMessageSendEmbed(s.channelID, &discordgo.MessageEmbed{
		Description: str,
	})
	if err != nil {
		log.Errorf("error sending message of length %v: %v", len(str), err)
	} else {
		log.Tracef("successfully sent message of length %v", len(str))
	}

	err = errors.Join(err, s.sendBlankLine())
	return msg, err
}

func (s *DiscordSender) sendBlankLine() error {
	_, err := s.session.ChannelMessageSend(s.channelID, settings.WhiteSpaceChar)
	if err != nil {
		log.Errorf("error sending blank line: %v", err)
	}

	return err
}

func (s *DiscordSender) addBQ(str string) string {
	parts := strings.Split(str, "\n")
	var output []string
	for _, s := range parts {
		if len(parts) == 0 {
			continue
		}

		output = append(output, "> "+s)
	}
	return strings.Join(output, "\n")
}
