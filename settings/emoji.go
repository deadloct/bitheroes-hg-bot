package settings

import "fmt"

type EmojiKey string

type EmojiInfo struct {
	Name     string
	ID       string
	Animated bool
}

func (e EmojiInfo) EmojiCode() string {
	if e.Name == "" || e.ID == "" {
		return ""
	}

	format := "<:%v:%v>"
	if e.Animated {
		format = "<a:%v:%v>"
	}

	return fmt.Sprintf(format, e.Name, e.ID)
}

var (
	EmojiParticipant EmojiKey = "Participant"
	EmojiClone       EmojiKey = "Clone"
	EmojiPresSnow    EmojiKey = "PresSnow"
	EmojiEffie       EmojiKey = "Effie"
	EmojiCaesar      EmojiKey = "Caesar"

	emojis = map[EmojiKey]EmojiInfo{
		EmojiParticipant: {
			Name:     GetenvStr("BITHEROES_HG_BOT_PARTICIPANT_EMOJI_NAME"),
			ID:       GetenvStr("BITHEROES_HG_BOT_PARTICIPANT_EMOJI_ID"),
			Animated: GetenvBool("BITHEROES_HG_BOT_PARTICIPANT_EMOJI_ANIMATED"),
		},

		EmojiClone: {
			Name:     GetenvStr("BITHEROES_HG_BOT_CLONE_EMOJI_NAME"),
			ID:       GetenvStr("BITHEROES_HG_BOT_CLONE_EMOJI_ID"),
			Animated: GetenvBool("BITHEROES_HG_BOT_CLONE_EMOJI_ANIMATED"),
		},

		EmojiPresSnow: {
			Name:     GetenvStr("BITHEROES_HG_BOT_PRESIDENT_SNOW_EMOJI_NAME"),
			ID:       GetenvStr("BITHEROES_HG_BOT_PRESIDENT_SNOW_EMOJI_ID"),
			Animated: GetenvBool("BITHEROES_HG_BOT_PRESIDENT_SNOW_EMOJI_ANIMATED"),
		},

		EmojiEffie: {
			Name:     GetenvStr("BITHEROES_HG_BOT_EFFIE_EMOJI_NAME"),
			ID:       GetenvStr("BITHEROES_HG_BOT_EFFIE_EMOJI_ID"),
			Animated: GetenvBool("BITHEROES_HG_BOT_EFFIE_EMOJI_ANIMATED"),
		},

		EmojiCaesar: {
			Name:     GetenvStr("BITHEROES_HG_BOT_CAESAR_EMOJI_NAME"),
			ID:       GetenvStr("BITHEROES_HG_BOT_CAESAR_EMOJI_ID"),
			Animated: GetenvBool("BITHEROES_HG_BOT_CAESAR_EMOJI_ANIMATED"),
		},
	}

	// Move to server
	// CloneEmojiName = "cloneparty"
	// CloneEmojiID   = "1091731392112627712"
	// PresSnowEmojiName = "presidentsnow"
	// PresSnowEmojiID   = "1091729861409775697"
)

func GetEmoji(key EmojiKey) EmojiInfo {
	v, ok := emojis[key]
	if !ok {
		return EmojiInfo{}
	}

	return v
}
