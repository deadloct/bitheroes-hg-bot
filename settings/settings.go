package settings

import (
	"text/template"
	"time"

	"github.com/deadloct/bitheroes-hg-bot/data"
	log "github.com/sirupsen/logrus"
)

const (
	// Seconds
	DefaultStartDelay = 10.0       // 10 minutes
	MinimumStartDelay = 0.5        // 30 seconds
	MaximumStartDelay = 4.0 * 60.0 // 4 hours

	EnableClone  = true
	DefaultClone = 1
	MinimumClone = 1
	MaximumClone = 20

	DefaultDayDelay    = 5 * time.Second
	DefaultVictorCount = 1
	MinimumVictorCount = 0

	JokeInterval = 10 * time.Second

	MaxMsgLen            = 1500
	DiscordMaxMessages   = 100
	DiscordMaxBulkDelete = 100

	WhiteSpaceChar = "\u200d"

	DataLocation = "data"
	DayEmoji     = "skull_crossbones"

	MaxQuietDays = 3
)

var (
	Intro *template.Template
	Help  string // not currently a template
)

type IntroValues struct {
	Delay       time.Duration
	EntryEmoji  string
	EffieEmoji  string
	CloneEmoji  string
	Clone       int
	MinimumTier int
	Sponsor     string
	VictorCount int
}

func ImportData() {
	importIntro()
	importHelp()
}

func importIntro() {
	var err error
	Intro, err = template.New("intro-template").Parse(data.IntroTemplate)
	if err != nil {
		log.Panicf("unable to parse intro template '%v': %v", data.IntroTemplate, err)
	}

	log.Info("imported intro template")
}

func importHelp() {
	Help = data.HelpTemplate
	log.Info("imported help file")
}
