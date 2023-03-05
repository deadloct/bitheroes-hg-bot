package stages

import "github.com/bwmarrin/discordgo"

type Stage interface {
	Run(session *discordgo.Session, msg *discordgo.Message, stop chan struct{}, users []*discordgo.User) []*discordgo.User
}
