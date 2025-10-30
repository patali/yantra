package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL             string
	JWTSecret               string
	Port                    string
	Environment             string
	AppURL                  string
	SystemEmailProvider     string // "smtp" or "resend"
	SystemEmailFrom         string
	SystemEmailFromName     string
	SystemEmailSMTPHost     string
	SystemEmailSMTPPort     string
	SystemEmailSMTPUser     string
	SystemEmailSMTPPassword string
	SystemEmailResendAPIKey string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		Port:                    getEnvOrDefault("PORT", "3000"),
		Environment:             getEnvOrDefault("NODE_ENV", "development"),
		AppURL:                  getEnvOrDefault("APP_URL", "http://localhost:3000"),
		SystemEmailProvider:     getEnvOrDefault("SYSTEM_EMAIL_PROVIDER", "smtp"),
		SystemEmailFrom:         getEnvOrDefault("SYSTEM_EMAIL_FROM", "noreply@yantra.local"),
		SystemEmailFromName:     getEnvOrDefault("SYSTEM_EMAIL_FROM_NAME", "Yantra"),
		SystemEmailSMTPHost:     os.Getenv("SYSTEM_EMAIL_SMTP_HOST"),
		SystemEmailSMTPPort:     getEnvOrDefault("SYSTEM_EMAIL_SMTP_PORT", "587"),
		SystemEmailSMTPUser:     os.Getenv("SYSTEM_EMAIL_SMTP_USER"),
		SystemEmailSMTPPassword: os.Getenv("SYSTEM_EMAIL_SMTP_PASSWORD"),
		SystemEmailResendAPIKey: os.Getenv("SYSTEM_EMAIL_RESEND_API_KEY"),
	}

	// Validate required config
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
