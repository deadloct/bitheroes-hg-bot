package data

type Mode struct {
	ID            string   `json:"id"`
	Name          string   `json:"mode"`
	IntroTemplate string   `json:"intro-template"`
	Rounds        []*Round `json:"rounds"`
}

type Round struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Intro     string   `json:"intro"`
	Survivors int      `json:"survivors"`
	Deaths    []string `json:"deaths"`
}
