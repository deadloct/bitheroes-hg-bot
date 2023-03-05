package rounds

type RedLightGreenLight struct {
}

func NewRedLightGreenLight() *RedLightGreenLight {
	return &RedLightGreenLight{}
}

func (rlgl *RedLightGreenLight) Run(contestants []Contestant, outputCh chan string) []Contestant {
	return nil
}
