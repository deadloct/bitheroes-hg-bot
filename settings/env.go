package settings

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const Prefix = "BITHEROES_HG_BOT"

func LoadEnvFiles() {
	env := os.Getenv(EnvKey("HG_BOT_ENV"))
	if env == "" {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")
	godotenv.Load(".env." + env)
	godotenv.Load()
}

func GetenvStr(key string) string {
	return os.Getenv(EnvKey(key))
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

func EnvKey(str string) string {
	return fmt.Sprintf("%s_%s", Prefix, str)
}
