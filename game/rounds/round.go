package rounds

type Contestant string

type Round interface {
	Run(contestants []Contestant, outputCh chan string) []Contestant
}
