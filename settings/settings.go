package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ParticipantEmoji     = "🕊️"
	ParticipantEmojiName = "dove"

	DefaultStartDelay = 60 * time.Second
	MinimumStartDelay = 5   // Seconds
	MaximumStartDelay = 900 // 15 minutes in seconds

	DayDelay = 5 * time.Second

	DataLocation = "data"
	PhrasesFile  = "phrases.en.json"
	IntroFile    = "intro.template"

	DiscordMaxMessageLength = 2000
	DiscordMaxMessages      = 100
	DiscordMaxBulkDelete    = 100

	DefaultSeparator = "_,.-'~'-.,__,.-'~'-.,_"
	HalfSeparator    = "_,.-'~'-.,_"
)

var (
	Intro   *template.Template
	Phrases []*template.Template
)

type IntroValues struct {
	User  string
	Delay time.Duration
}

type PhraseValues struct {
	Dying  string
	Killer string
}

func ImportData() {
	importIntro()
	importPhrases()
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

func importPhrases() {
	phrasesFilePath := path.Join(DataLocation, PhrasesFile)
	data, err := ioutil.ReadFile(phrasesFilePath)
	if err != nil {
		log.Panicf("could not read the phrases file %v: %v", phrasesFilePath, err)
	}

	var phraseStrings []string
	if err := json.Unmarshal(data, &phraseStrings); err != nil {
		log.Panicf("could not parse the phrases file %v: %v", phrasesFilePath, err)
	}

	if len(phraseStrings) == 0 {
		log.Panicf("there are no phrases in the phrases file %v", phrasesFilePath)
	}

	for i, phrase := range phraseStrings {
		phraseTmpl, err := template.New(fmt.Sprintf("phrase-%v", i)).Parse(phrase)
		if err != nil {
			log.Panicf("unable to parse phrase '%v': %v", phrase, err)
		}

		Phrases = append(Phrases, phraseTmpl)
	}

	if len(Phrases) == 0 {
		log.Panicf("no phrase templates were parsed")
	}

	log.Infof("imported %v phrases", len(Phrases))
}
