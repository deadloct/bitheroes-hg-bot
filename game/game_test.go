package game

import (
	"fmt"
	"html/template"
	"sync"
	"testing"

	"github.com/deadloct/bitheroes-hg-bot/settings"

	"github.com/bwmarrin/discordgo"
)

func TestGame_getRandomPhrase_SingleReplace(t *testing.T) {
	testSetupPhrases(t, []string{"{{.Dying}} killed by {{.Killer}}"})

	dying := &discordgo.User{Username: "dying user"}
	living := []*discordgo.User{
		{Username: "living 1"},
		{Username: "living 2"},
	}

	s := &BufferSender{}
	s.Start()
	defer s.Stop()

	actual := NewGame(GameConfig{Sender: s}).getRandomPhrase(dying, living)
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

	s := &BufferSender{}
	s.Start()
	defer s.Stop()

	actual := NewGame(GameConfig{Sender: s}).getRandomPhrase(dying, living)
	expected1 := fmt.Sprintf(phrase, living[0].Username, dying.Username, dying.Username, living[0].Username)
	expected2 := fmt.Sprintf(phrase, living[1].Username, dying.Username, dying.Username, living[1].Username)
	if actual != expected1 && actual != expected2 {
		t.Errorf("expected '%v' to equal '%v' or '%v'", actual, expected1, expected2)
	}
}

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

type BufferSender struct {
	buffer []string
	msgCh  chan string
	stopCh chan struct{}
	sync.Mutex
}

func (b *BufferSender) Start() chan string {
	b.msgCh = make(chan string)
	b.stopCh = make(chan struct{})
	return b.msgCh
}

func (b *BufferSender) listen() {
	for {
		select {
		case msg := <-b.msgCh:
			b.Lock()
			b.buffer = append(b.buffer, msg)
			b.Unlock()
		case <-b.stopCh:
			return
		}
	}
}

func (b *BufferSender) Stop() {
	close(b.stopCh)
}

func (b *BufferSender) Send(str string) (*discordgo.Message, error) {
	b.Lock()
	defer b.Unlock()
	b.buffer = append(b.buffer, str)
	return nil, nil
}
