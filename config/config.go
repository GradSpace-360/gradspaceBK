package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfig() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
}

func GetResendAPIKey() string {
    apiKey := os.Getenv("RESEND_API_KEY")
    return apiKey
}