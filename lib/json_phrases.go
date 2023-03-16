package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type PhraseValues struct {
	Dying  string
	Killer string
}

type JSONPhrases struct {
	templateIndexes []int
	templates       []*template.Template
}

func NewJSONPhrases(data []byte) *JSONPhrases {
	o := &JSONPhrases{}
	o.importJSON(data)
	o.generateTemplateIndexes()
	return o
}

func (jp *JSONPhrases) GetRandomPhrase(user string, mention string, living []string) string {
	defaultPhrase := fmt.Sprintf("%v died of dysentery.", user)

	killer := "another player"
	if killerNum, err := GetRandomInt(0, len(living)); err == nil {
		killer = living[killerNum]
	}

	i, err := GetRandomInt(0, len(jp.templateIndexes))
	if err != nil {
		log.Errorf("could not retrieve random int for picking a phrase: %v", err)
		return defaultPhrase
	}

	dyingName := fmt.Sprintf("**%v**", user)
	if mention != "" {
		dyingName = mention
	}

	var result bytes.Buffer
	tmpl := jp.templates[jp.templateIndexes[i]]
	vals := PhraseValues{
		Killer: killer,
		Dying:  dyingName,
	}
	if err := tmpl.Execute(&result, vals); err != nil {
		log.Errorf("error executing template with vals: %v", err)
		return defaultPhrase
	}

	if len(jp.templateIndexes) == 1 {
		jp.generateTemplateIndexes()
	} else {
		jp.templateIndexes = append(jp.templateIndexes[:i], jp.templateIndexes[i+1:]...)
	}

	return result.String()
}

func (jp *JSONPhrases) PhraseCount() int {
	return len(jp.templates)
}

func (jp *JSONPhrases) importJSON(data []byte) {
	var phraseStrings []string
	if err := json.Unmarshal(data, &phraseStrings); err != nil {
		log.Panicf("could not parse the phrases data %v: %v", data, err)
	}

	if len(phraseStrings) == 0 {
		log.Panicf("there are no phrases in the phrases data %v", data)
	}

	for i, phrase := range phraseStrings {
		phraseTmpl, err := template.New(fmt.Sprintf("phrase-%v", i)).Parse(phrase)
		if err != nil {
			log.Panicf("unable to parse phrase '%v': %v", phrase, err)
		}

		jp.templates = append(jp.templates, phraseTmpl)
	}

	if len(jp.templates) == 0 {
		log.Panicf("no phrase templates were parsed")
	}
}

func (jp *JSONPhrases) generateTemplateIndexes() {
	n := len(jp.templates)
	jp.templateIndexes = make([]int, n)
	for i := 0; i < n; i++ {
		jp.templateIndexes[i] = i
	}
}
