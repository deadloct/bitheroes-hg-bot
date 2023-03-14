package settings

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPhrases(t *testing.T) {
	oldDataLocation := DataLocation
	defer func() {
		DataLocation = oldDataLocation
	}()

	DataLocation = fmt.Sprintf("../%s", DataLocation)

	importPhrases()

	for i, phrase := range Phrases {
		p := phrase
		t.Run(fmt.Sprintf("Template %v", i), func(t *testing.T) {
			t.Parallel()
			var result bytes.Buffer
			err := p.Execute(&result, PhraseValues{
				Dying:  "dying-user",
				Killer: "killer-user",
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
}
