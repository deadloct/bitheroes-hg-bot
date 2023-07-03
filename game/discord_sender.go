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
	SendQuoted(str string) (*discordgo.Message, error)
	SendEmbed(str string) (*discordgo.Message, error)
	SendDM(user *discordgo.User, msg string) error
}

type SendingFunc func(str string) (*discordgo.Message, error)

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

func (s *DiscordSender) SendQuoted(str string) (*discordgo.Message, error) {
	return s.send(str, s.flushQuoted)
}

func (s *DiscordSender) SendEmbed(str string) (*discordgo.Message, error) {
	return s.send(str, s.flushEmbed)
}

func (s *DiscordSender) SendDM(user *discordgo.User, msg string) error {
	dmChannel, err := s.session.UserChannelCreate(user.ID)
	if err != nil {
		return err
	}

	if _, err = s.session.ChannelMessageSend(dmChannel.ID, msg); err != nil {
		return err
	}

	return nil
}

func (s *DiscordSender) send(str string, sender SendingFunc) (*discordgo.Message, error) {
	lines := strings.Split(str, "\n")

	var (
		msg     *discordgo.Message
		errs    []error
		err     error
		payload string
	)

	for i := 0; i < len(lines); i++ {
		nextlen := len(payload) + len(lines[i]) + 1
		if nextlen < settings.MaxMsgLen {
			switch payload {
			case "":
				payload = lines[i]
			default:
				payload = fmt.Sprintf("%s\n%s", payload, lines[i])
			}

			continue
		}

		if payload != "" {
			// a blankline will be sent in between messages anyway, so don't double up.
			lastline := lines[i-1]
			if lastline == settings.WhiteSpaceChar {
				payload = strings.TrimSuffix(payload, fmt.Sprintf("\n%s", settings.WhiteSpaceChar))
			}

			payload = strings.TrimSuffix(payload, "\n")
			_, err = s.sendLine(payload, sender)
			errs = append(errs, err)
		}

		payload = lines[i]
	}

	if payload != "" {
		msg, err = s.sendLine(payload, sender)
		errs = append(errs, err)
	}

	return msg, errors.Join(errs...)
}

func (s *DiscordSender) sendLine(str string, sender SendingFunc) (*discordgo.Message, error) {
	if len(str) <= settings.MaxMsgLen {
		return sender(str)
	}

	words := strings.Fields(str)
	var (
		line string
		msg  *discordgo.Message
		err  error
	)

	for _, word := range words {
		// Space character between line and word is why this uses >= instead of >
		if len(line)+len(word)+1 >= settings.MaxMsgLen {
			if msg, err = sender(line); err != nil {
				return nil, err
			}

			line = ""
		}

		if line == "" {
			line = word
		} else {
			line = fmt.Sprintf("%s %s", line, word)
		}
	}

	if len(line) > 0 {
		if msg, err = sender(line); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (s *DiscordSender) flushQuoted(str string) (*discordgo.Message, error) {
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

func (s *DiscordSender) flushEmbed(str string) (*discordgo.Message, error) {
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
