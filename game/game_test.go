package game

import (
	"fmt"
	"html/template"
	"testing"

	"github.com/deadloct/bitheroes-hg-bot/settings"

	"github.com/bwmarrin/discordgo"
)

func testSetupPhrases(t *testing.T, testPhrases []string) {
	t.Helper()

	settings.Phrases = nil
	for i, phrase := range testPhrases {
		tmpl, err := template.New(fmt.Sprintf("phrase-%v", i)).Parse(phrase)
		if err != nil {
			t.Fatalf("error loading test phrase '%v': %v", phrase, err)
		}

		settings.Phrases = append(settings.Phrases, tmpl)
	}
}

func TestGame_getRandomPhrase_SingleReplace(t *testing.T) {
	testSetupPhrases(t, []string{"{{.Dying}} killed by {{.Killer}}"})

	dying := &discordgo.User{Username: "dying user"}
	living := []*discordgo.User{
		{Username: "living 1"},
		{Username: "living 2"},
	}

	actual := NewGame().getRandomPhrase(dying, living)
	expected1 := fmt.Sprintf("%v killed by %v", dying.Username, living[0].Username)
	expected2 := fmt.Sprintf("%v killed by %v", dying.Username, living[1].Username)
	if actual != expected1 && actual != expected2 {
		t.Errorf("expected '%v' to equal '%v' or '%v'", actual, expected1, expected2)
	}
}

func TestGame_getRandomPhrase_MultiReplace(t *testing.T) {
	phrase := "%s gave %s a poison flower, and %s said thanks while %s laughed"
	testSetupPhrases(t, []string{
		fmt.Sprintf(phrase, "{{.Killer}}", "{{.Dying}}", "{{.Dying}}", "{{.Killer}}"),
	})

	dying := &discordgo.User{Username: "dying user"}
	living := []*discordgo.User{
		{Username: "living 1"},
		{Username: "living 2"},
	}

	actual := NewGame().getRandomPhrase(dying, living)
	expected1 := fmt.Sprintf(phrase, living[0].Username, dying.Username, dying.Username, living[0].Username)
	expected2 := fmt.Sprintf(phrase, living[1].Username, dying.Username, dying.Username, living[1].Username)
	if actual != expected1 && actual != expected2 {
		t.Errorf("expected '%v' to equal '%v' or '%v'", actual, expected1, expected2)
	}
}
