package game

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

type Jester struct {
	generator JokeGenerator
	sender    Sender
	session   *discordgo.Session
}

func NewJester(jg JokeGenerator, s Sender, sess *discordgo.Session) *Jester {
	return &Jester{generator: jg, sender: s, session: sess}
}

func (j *Jester) StartRandomJokes(ctx context.Context, stop chan struct{}) {
	if j.generator == nil {
		log.Warn("not starting joke engine because joke generator is nil")
		return
	}

	go func() {
		ticker := time.NewTicker(settings.JokeInterval)
		time.Sleep(2 * time.Second)
		msg, err := j.sendJoke(nil)
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
				if msg, err = j.sendJoke(msg); err != nil {
					return
				}
			}
		}
	}()
}

func (j *Jester) sendJoke(msg *discordgo.Message) (*discordgo.Message, error) {
	intro := "Greetings, esteemed guests and citizens of the Capitol! As the royal jester to our illustrious President Snow, I stand before you today to bring some much-needed levity and humor to this esteemed gathering. I understand that some of you may be feeling impatient, but fear not! I am here to entertain you with the finest collection of dad jokes this side of the Districts."

	joke, err := j.generator.GetJoke()
	if err != nil {
		log.Warnf("failed to get joke: %v", err)
		return msg, err
	}

	text := fmt.Sprintf("> %v\n> %v\n> \n> *%v*\n> *%v*\n> %v", settings.BlankLine, intro, joke.Question, joke.Answer, settings.BlankLine)
	if msg == nil {
		if msg, err = j.sender.Send(text); err != nil {
			log.Warnf("failed to send joke: %v", err)
			return msg, err
		}
	} else {
		if msg, err = j.session.ChannelMessageEdit(msg.ChannelID, msg.ID, text); err != nil {
			log.Warnf("failed to edit joke: %v", err)
			return msg, err
		}
	}

	return msg, nil
}
