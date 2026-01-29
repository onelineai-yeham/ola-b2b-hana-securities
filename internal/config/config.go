package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	Silver DBConfig
	Gold   DBConfig
	Batch  BatchConfig
}

type ServerConfig struct {
	Port     int
	LogLevel string
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	Schema   string
}

type BatchConfig struct {
	Interval time.Duration
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?search_path=%s&sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Name, d.Schema,
	)
}

func Load() (*Config, error) {
	// Load .env.local first (local development), then .env as fallback
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	cfg := &Config{}

	// Server config
	cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
	cfg.Server.LogLevel = getEnv("LOG_LEVEL", "info")

	// Silver DB config
	cfg.Silver.Host = getEnv("SILVER_DB_HOST", "localhost")
	cfg.Silver.Port = getEnvAsInt("SILVER_DB_PORT", 5432)
	cfg.Silver.Name = getEnv("SILVER_DB_NAME", "etl")
	cfg.Silver.User = getEnv("SILVER_DB_USER", "")
	cfg.Silver.Password = getEnv("SILVER_DB_PASSWORD", "")
	cfg.Silver.Schema = getEnv("SILVER_DB_SCHEMA", "silver")

	// Gold DB config
	cfg.Gold.Host = getEnv("GOLD_DB_HOST", "localhost")
	cfg.Gold.Port = getEnvAsInt("GOLD_DB_PORT", 5432)
	cfg.Gold.Name = getEnv("GOLD_DB_NAME", "hana_securities")
	cfg.Gold.User = getEnv("GOLD_DB_USER", "")
	cfg.Gold.Password = getEnv("GOLD_DB_PASSWORD", "")
	cfg.Gold.Schema = getEnv("GOLD_DB_SCHEMA", "gold")

	// Batch config
	intervalMinutes := getEnvAsInt("BATCH_INTERVAL_MINUTES", 10)
	cfg.Batch.Interval = time.Duration(intervalMinutes) * time.Minute

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
