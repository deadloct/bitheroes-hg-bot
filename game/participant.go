package game

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Participant struct {
	*discordgo.Member

	AlternateDisplayName string
}

func NewParticipant(m *discordgo.Member) *Participant {
	return &Participant{Member: m}
}

func (p *Participant) DisplayName() string {
	if p.AlternateDisplayName != "" {
		return p.AlternateDisplayName
	}

	if p.Nick != "" {
		return p.Nick
	}

	return p.User.Username
}

func (p *Participant) Mention() string {
	return fmt.Sprintf("<@%v>", p.User.ID)
}

func (p *Participant) DisplayFullName() string {
	displayName := p.User.Username

	if p.Nick != "" {
		displayName = p.Nick
	}

	return fmt.Sprintf("%v (%v#%v)", displayName, p.User.Username, p.User.Discriminator)
}
