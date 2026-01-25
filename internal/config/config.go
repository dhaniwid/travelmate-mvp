package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DBUrl        string
	OpenAIKey    string
	AllowOrigins string
	GoogleAPIKey string
	GoogleCXId   string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	dbUrl := os.Getenv("DATABASE_URL")

	if dbUrl == "" {
		log.Println("⚠️ DATABASE_URL not set")
	}

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DBUrl:        dbUrl,
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		AllowOrigins: "*",
		GoogleAPIKey: os.Getenv("GOOGLE_API_KEY"),
		GoogleCXId:   os.Getenv("GOOGLE_CX_ID"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
