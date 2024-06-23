package settings

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

const Prefix = "BITHEROES_HG_BOT"

func LoadEnvFiles() {
	env := os.Getenv(EnvKey("HG_BOT_ENV"))
	if env == "" {
		env = "development"
	}

	godotenv.Load(envPath(fmt.Sprintf(".env.%s.local", env)))
	godotenv.Load(envPath(fmt.Sprintf(".env.%s", env)))
	godotenv.Load(envPath(".env"))
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

func envPath(filename string) string {
	p, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	p = filepath.Dir(p)
	return path.Join(p, filename)
}
