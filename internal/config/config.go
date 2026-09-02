package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	KafkaBrokers string
	VaultKey     string // 32-byte key for AES-256
}

// Load loads configuration from environment variables or .env file
func Load() *Config {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	cfg := &Config{
		Port:         getEnv("PORT", "8082"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://pp_user:pp_secret@localhost:5432/payment_processor?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		VaultKey:     getEnv("VAULT_ENCRYPTION_KEY", "12345678901234567890123456789012"), // Default for dev ONLY
	}

	if len(cfg.VaultKey) != 32 {
		log.Fatalf("VAULT_ENCRYPTION_KEY must be exactly 32 bytes")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
