package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"
)

const (
	CMD_PREFIX = "!"
	CMD_HG     = CMD_PREFIX + "hg"

	DefaultStartDelay = 5 * time.Second
	MinimumStartDelay = 30  // Seconds
	MaximumStartDelay = 600 // 10 minutes in seconds

	ParticipantEmoji = "🕊️" // :dove:
	SettingsLocation = "data"
)

type Settings struct {
	IntroTemplate string   `json:"intro"`
	Phrases       []string `json:"phrases"`
}

func LoadJSONData() (*Settings, error) {
	data, err := ioutil.ReadFile("./data/settings.en.json")
	if err != nil {
		return nil, err
	}

	var settings *Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	if len(settings.Phrases) == 0 {
		return nil, errors.New("no phrases to load, exiting...")
	}

	return settings, nil
}
