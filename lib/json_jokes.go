package lib

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

type Joke struct {
	Question string `json:"q"`
	Answer   string `json:"a"`
}

type JSONJokes struct {
	jokes   []Joke
	indexes []int
}

func NewJSONJokes(data []byte) (*JSONJokes, error) {
	var jokes []Joke
	if err := json.Unmarshal(data, &jokes); err != nil {
		return nil, err
	}

	log.Debugf("created new jokes generator with %v jokes", len(jokes))
	generator := &JSONJokes{jokes: jokes}
	generator.generateJokeIndexes()
	return generator, nil
}

func (jj *JSONJokes) GetJoke() (Joke, error) {
	var joke Joke
	i, err := GetRandomInt(0, len(jj.indexes))
	if err != nil {
		log.Errorf("could not retrieve random int for picking a phrase: %v", err)
		return joke, err
	}

	joke = jj.jokes[i]

	if len(jj.indexes) == 1 {
		jj.generateJokeIndexes()
	} else {
		jj.indexes = append(jj.indexes[:i], jj.indexes[i+1:]...)
	}

	return joke, nil
}

func (jj *JSONJokes) generateJokeIndexes() {
	n := len(jj.jokes)
	jj.indexes = make([]int, n)
	for i := 0; i < n; i++ {
		jj.indexes[i] = i
	}
}
