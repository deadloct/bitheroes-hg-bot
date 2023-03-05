package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	CMD_PREFIX = "!"
	CMD_HG     = CMD_PREFIX + "hg"

	ParticipantEmoji = "🕊️" // :dove:

	DefaultStartDelay = 5 * time.Second
	MinimumStartDelay = 30  // Seconds
	MaximumStartDelay = 600 // 10 minutes in seconds

	DayDelay = 5 * time.Second

	DataLocation = "data"
	PhrasesFile  = "phrases.en.json"
	IntroFile    = "intro.template"

	DiscordMaxMessageLength = 2000
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

func init() {
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
