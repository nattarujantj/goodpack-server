package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	MongoURI    string
	Database    string
	Environment string

	JWTSecret string
	JWTExpiry string

	AdminUsername    string
	AdminPassword    string
	AdminDisplayName string
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		Database:    getEnv("DATABASE_NAME", "goodpack"),
		Environment: getEnv("ENVIRONMENT", "development"),

		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry: getEnv("JWT_EXPIRY", "24h"),

		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		AdminDisplayName: getEnv("ADMIN_DISPLAY_NAME", "Super Admin"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
