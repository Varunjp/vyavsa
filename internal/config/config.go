package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseURL string

	RedisAddr string
	RedisDB   int

	LogLevel    string
	ServiceName string
}

func Load() Config {
	_ = godotenv.Load()
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		DatabaseURL: getEnv(
			"DATABASE_URL",
			"postgres://postgres:postgres@localhost:5432/billbook",
		),

		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:   redisDB,

		LogLevel: getEnv(
			"LOG_LEVEL",
			"info",
		),

		ServiceName: getEnv(
			"SERVICE_NAME",
			"bill-book-api",
		),
	}

}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
