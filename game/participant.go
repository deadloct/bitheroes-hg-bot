package game

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Participant struct {
	*discordgo.Member
}

func NewParticipant(m *discordgo.Member) *Participant {
	return &Participant{Member: m}
}

func (p *Participant) DisplayName() string {
	if p.Nick != "" {
		return p.Nick
	}

	return p.User.Username
}

func (p *Participant) Mention() string {
	return fmt.Sprintf("<@%v>", p.User.ID)
}
