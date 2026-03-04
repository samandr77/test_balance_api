package config

import (
	"log"
	"os"
)

type Config struct {
	DatabaseURL string
	APIToken    string
	Port        string
	AppEnv      string
}

func Load() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		log.Fatal("API_TOKEN is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	return Config{
		DatabaseURL: dbURL,
		APIToken:    apiToken,
		Port:        port,
		AppEnv:      appEnv,
	}
}
