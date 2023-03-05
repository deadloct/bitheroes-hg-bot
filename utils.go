package main

import (
	"crypto/rand"
	"math/big"
)

func GetRandomInt(min, max int) (int, error) {
	bg := big.NewInt(int64(max - min))
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		return 0, err
	}

	return int(n.Int64()) + min, nil
}
