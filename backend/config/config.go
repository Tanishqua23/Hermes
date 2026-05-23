package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresDSN        string
	RedisAddr          string
	APIPort            string
	WorkerConcurrency  int
}

func Load() *Config {
	if err := godotenv.Load(".env"); err != nil  {
		// .env optional in production
		log.Println("no .env file found, reading environment")
	}

	concurrency := 5
	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			concurrency = n
		}
	}

	return &Config{
		PostgresDSN:       getEnv("POSTGRES_DSN", "postgres://hermes:hermes_secret@localhost:5432/hermes?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		APIPort:           getEnv("API_PORT", "8080"),
		WorkerConcurrency: concurrency,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
