package stages

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RLGLName         = "Round 1: Red Light, Green Light."
	RLGLIntro        = "♫ Mugunghwa Kkoci Pieot Seumnida ( 무궁화 꽃 이 피었 습니다 ) ♫"
	RLGLMaxSurvivors = 60
	RedLightDuration = 2 * time.Second
)

type RedLightGreenLight struct {
}

func NewRedLightGreenLight() *RedLightGreenLight {
	return &RedLightGreenLight{}
}

func (rlgl *RedLightGreenLight) Run(session *discordgo.Session, msg *discordgo.Message, stop chan struct{}, users []*discordgo.User) []*discordgo.User {
	session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("**%v**\n\n%v", RLGLName, RLGLIntro))

	// TODO: Actually make this stage
	// var killed []int64
	// killed = append(killed, lib.GetRandomInt(0, int64(len(users))))

	return users
}
