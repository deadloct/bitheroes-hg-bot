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

	emojis map[EmojiKey]EmojiInfo
)

func LoadEmojis() {
	emojis = map[EmojiKey]EmojiInfo{
		EmojiParticipant: {
			Name:     GetenvStr("PARTICIPANT_EMOJI_NAME"),
			ID:       GetenvStr("PARTICIPANT_EMOJI_ID"),
			Animated: GetenvBool("PARTICIPANT_EMOJI_ANIMATED"),
		},

		EmojiClone: {
			Name:     GetenvStr("CLONE_EMOJI_NAME"),
			ID:       GetenvStr("CLONE_EMOJI_ID"),
			Animated: GetenvBool("CLONE_EMOJI_ANIMATED"),
		},

		EmojiPresSnow: {
			Name:     GetenvStr("PRESIDENT_SNOW_EMOJI_NAME"),
			ID:       GetenvStr("PRESIDENT_SNOW_EMOJI_ID"),
			Animated: GetenvBool("PRESIDENT_SNOW_EMOJI_ANIMATED"),
		},

		EmojiEffie: {
			Name:     GetenvStr("EFFIE_EMOJI_NAME"),
			ID:       GetenvStr("EFFIE_EMOJI_ID"),
			Animated: GetenvBool("EFFIE_EMOJI_ANIMATED"),
		},

		EmojiCaesar: {
			Name:     GetenvStr("CAESAR_EMOJI_NAME"),
			ID:       GetenvStr("CAESAR_EMOJI_ID"),
			Animated: GetenvBool("CAESAR_EMOJI_ANIMATED"),
		},
	}
}

func GetEmoji(key EmojiKey) EmojiInfo {
	v, ok := emojis[key]
	if !ok {
		return EmojiInfo{}
	}

	return v
}
