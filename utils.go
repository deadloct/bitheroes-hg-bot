package main

import (
	"crypto/rand"
	"math/big"
)

// Source: https://stackoverflow.com/a/26153749
func GetRandomInt(min, max int64) int64 {
	bg := big.NewInt(max - min)
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err)
	}

	return n.Int64() + min
}
