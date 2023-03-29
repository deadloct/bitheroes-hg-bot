package settings

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// Seconds
	DefaultStartDelay = 60
	MinimumStartDelay = 5
	MaximumStartDelay = 60 * 60 // 60 minutes in seconds

	EnableEntryMultiplier  = true
	DefaultEntryMultiplier = 1
	MinimumEntryMultiplier = 1
	MaximumEntryMultiplier = 100

	DefaultDayDelay    = 5 * time.Second
	DefaultVictorCount = 1
	MinimumVictorCount = 0

	JokeInterval = 10 * time.Second

	PhrasesFile = "phrases.en.json"
	JokesFile   = "jokes.en.json"
	IntroFile   = "intro.en.template"
	HelpFile    = "help.en.template"

	DiscordMaxMessageLength = 2000
	DiscordMaxMessages      = 100
	DiscordMaxBulkDelete    = 100

	DefaultSeparator = "_,.-'~'-.,__,.-'~'-.,_"
	HalfSeparator    = "_,.-'~'-.,_"
)

var (
	Intro *template.Template
	Help  string // not currently a template

	DataLocation         = "data"
	ParticipantEmojiName = os.Getenv("BITHEROES_HG_BOT_EMOJI_NAME")
	ParticipantEmojiID   = os.Getenv("BITHEROES_HG_BOT_EMOJI_ID")
	ParticipantEmojiCode = fmt.Sprintf("<:%v:%v>", ParticipantEmojiName, ParticipantEmojiID)
)

type IntroValues struct {
	Delay       time.Duration
	EmojiCode   string
	User        string
	VictorCount int
}

func ImportData() {
	importIntro()
	importHelp()
}

func importIntro() {
	introPath := path.Join(DataLocation, IntroFile)
	v, err := ioutil.ReadFile(introPath)
	if err != nil {
		log.Panicf("unable to open intro template file %v: %v", introPath, err)
	}

	tmplStr := string(v[:])
	Intro, err = template.New("intro-template").Parse(tmplStr)
	if err != nil {
		log.Panicf("unable to parse intro template '%v': %v", tmplStr, err)
	}

	log.Info("imported intro template")
}

func importHelp() {
	helpPath := path.Join(DataLocation, HelpFile)
	v, err := ioutil.ReadFile(helpPath)
	if err != nil {
		log.Panicf("unable to open help file %v: %v", helpPath, err)
	}

	Help = string(v[:])
	log.Info("imported help file")
}
