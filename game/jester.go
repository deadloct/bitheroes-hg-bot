package game

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
	log "github.com/sirupsen/logrus"
)

const MaxJokeFailures = 5

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
		defer ticker.Stop()

		time.Sleep(2 * time.Second)
		msg, err := j.sendJoke(nil)
		if err != nil {
			return
		}

		var failures int
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				if msg, err = j.sendJoke(msg); err != nil {
					if failures == MaxJokeFailures {
						log.Errorf("failed to send joke %v times, ending the Jester", MaxJokeFailures)
						return
					}

					log.Warn("failed to send joke, waiting for next ticker")
					failures++
				} else {
					failures = 0
				}
			}
		}
	}()
}

func (j *Jester) sendJoke(msg *discordgo.Message) (*discordgo.Message, error) {
	caesarEmoji := settings.GetEmoji(settings.EmojiCaesar)
	intro := fmt.Sprintf("%v  %v",
		caesarEmoji.EmojiCode(),
		"Greetings, esteemed guests and citizens of the Capitol! I am Caesar Flickerman, the host of this year's Hunger Games! What a time to be alive! I stand before you today to bring some much-needed levity and humor to this esteemed gathering. I understand that some of you may be feeling impatient, but fear not! I am here to entertain you with the finest collection of fatherly quips this side of the Districts.",
	)

	joke, err := j.generator.GetJoke()
	if err != nil {
		log.Warnf("failed to get joke: %v", err)
		return msg, err
	}

	text := fmt.Sprintf("%v\n\n*%v*\n*%v*", intro, joke.Question, joke.Answer)
	if msg == nil {
		if msg, err = j.sender.SendQuoted(text); err != nil {
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
