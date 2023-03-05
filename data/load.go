package data

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

func LoadJSONData() ([]*Mode, error) {
	data, err := ioutil.ReadFile("./data/data.en.json")
	if err != nil {
		return nil, err
	}

	var modes []*Mode
	if err := json.Unmarshal(data, &modes); err != nil {
		return nil, err
	}

	if len(modes) == 0 {
		return nil, errors.New("no modes to load, exiting...")
	}

	return modes, nil
}
