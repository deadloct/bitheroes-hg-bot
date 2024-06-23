package lib

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/deadloct/bitheroes-hg-bot/settings"
)

func TestJSONPhrases_GetRandomPhrase_AllCompileAndExec(t *testing.T) {
	data, err := os.ReadFile(path.Join("..", settings.DataLocation, "phrases.en.json"))
	if err != nil {
		t.Fatal(err)
	}

	jp := NewJSONPhrases(data)
	for i, phrase := range jp.templates {
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

func TestJSONPhrases_GetRandomPhrase_AllThenReset(t *testing.T) {
	data, err := os.ReadFile(path.Join("..", settings.DataLocation, "phrases.en.json"))
	if err != nil {
		t.Fatal(err)
	}

	jp := NewJSONPhrases(data)
	phraseCount := jp.PhraseCount()
	seen := make(map[string]int, phraseCount)

	if phraseCount == 0 {
		t.Fatal("there should be more than 0 phrases")
	}

	for i := 0; i < phraseCount; i++ {
		str := jp.GetRandomPhrase("hey", "<@hey>", []string{"yo"})
		if _, ok := seen[str]; ok {
			t.Fatalf("first round - dupe phrase before all have been used ('%v')", str)
		}

		seen[str] = 1
	}

	if len(seen) != phraseCount {
		t.Fatalf("should have seen %v phrases instead of %v after first round", phraseCount, len(seen))
	}

	for i := 0; i < phraseCount; i++ {
		str := jp.GetRandomPhrase("hey", "<@hey>", []string{"yo"})
		if seen[str] > 1 {
			t.Fatalf("second round - phrase used again before all have been used ('%v')", str)
		}

		seen[str]++
	}

	if len(seen) != phraseCount {
		t.Fatalf("should have seen %v phrases instead of %v after second round", phraseCount, len(seen))
	}
}

func TestJSONPhrases_GetRandomPhrase_SingleReplace(t *testing.T) {
	data := []byte(`["{{.Dying}} killed by {{.Killer}}"]`)
	jp := NewJSONPhrases(data)

	dying := &discordgo.User{Username: "dying user", ID: "123"}
	dyingMention := fmt.Sprintf("<@%v>", dying.ID)
	living := []string{"Player 1", "Player 2"}

	actual := jp.GetRandomPhrase(dying.Username, dyingMention, living)
	expected1 := fmt.Sprintf("%v killed by %v", dyingMention, living[0])
	expected2 := fmt.Sprintf("%v killed by %v", dyingMention, living[1])
	if actual != expected1 && actual != expected2 {
		t.Errorf("expected '%v' to equal '%v' or '%v'", actual, expected1, expected2)
	}
}

func TestGame_getRandomPhrase_MultiReplace(t *testing.T) {
	phrase := "%s gave %s a poison flower, and %s said thanks while %s laughed"
	data := []byte(fmt.Sprintf(`["%s"]`, fmt.Sprintf(phrase, "{{.Killer}}", "{{.Dying}}", "{{.Dying}}", "{{.Killer}}")))
	jp := NewJSONPhrases(data)

	dying := &discordgo.User{Username: "dying user", ID: "123"}
	dyingMention := fmt.Sprintf("<@%v>", dying.ID)
	living := []string{"Player 1", "Player 2"}

	actual := jp.GetRandomPhrase(dying.Username, dyingMention, living)
	expected1 := fmt.Sprintf(phrase, living[0], dyingMention, dyingMention, living[0])
	expected2 := fmt.Sprintf(phrase, living[1], dyingMention, dyingMention, living[1])
	if actual != expected1 && actual != expected2 {
		t.Errorf("expected '%v' to equal '%v' or '%v'", actual, expected1, expected2)
	}
}
