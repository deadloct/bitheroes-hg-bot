package game

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

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
