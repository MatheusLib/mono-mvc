package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr   string
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Addr:   getEnv("APP_ADDR", ":8080"),
		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "3306"),
		DBUser: getEnv("DB_USER", "admin"),
		DBPass: getEnv("DB_PASS", ""),
		DBName: getEnv("DB_NAME", "tcc"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
