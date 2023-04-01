package settings

import (
	"os"
	"strconv"
)

var AuthToken = GetenvStr("BITHEROES_HG_BOT_AUTH_TOKEN")

func GetenvStr(key string) string {
	return os.Getenv(key)
}

func GetenvInt(key string) int {
	s := GetenvStr(key)
	if s == "" {
		return 0
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return v
}

func GetenvBool(key string) bool {
	s := GetenvStr(key)
	if s == "" {
		return false
	}

	v, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}

	return v
}
