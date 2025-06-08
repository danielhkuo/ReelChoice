package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration values for the application
type Config struct {
	Port        string
	RedisURL    string
	DatabaseURL string
	TMDBApiKey  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Try to load .env file (ignore error if file doesn't exist in production)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	config := &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		RedisURL:    getEnvOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		DatabaseURL: getEnvOrDefault("DATABASE_URL", ""),
		TMDBApiKey:  getEnvOrDefault("TMDB_API_KEY", ""),
	}

	// Validate required configuration
	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	if config.TMDBApiKey == "" {
		log.Fatal("TMDB_API_KEY environment variable is required")
	}

	return config
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
