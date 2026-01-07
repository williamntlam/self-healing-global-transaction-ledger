package config

import (
	"os"
	"strconv"
)

// Config holds all non-sensitive configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	AWS      AWSConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Port   int
	Region string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
}

// AWSConfig holds AWS/LocalStack configuration
type AWSConfig struct {
	Region   string
	Endpoint string
	S3Bucket  string
	SQSQueue  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	return Config{
		App: AppConfig{
			Port:   getEnvInt("APP_PORT", 8080),
			Region: getEnv("REGION", "us-east-1"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("COCKROACHDB_HOST", "cockroachdb-public"),
			Port:     getEnvInt("COCKROACHDB_PORT", 26257),
			Database: getEnv("COCKROACHDB_DATABASE", "ledger"),
		},
		AWS: AWSConfig{
			Region:   getEnv("AWS_REGION", "us-east-1"),
			Endpoint: getEnv("AWS_ENDPOINT", "http://localhost:4566"),
			S3Bucket: getEnv("S3_BUCKET", "us-east-1-audit-logs"),
			SQSQueue: getEnv("SQS_QUEUE", "us-east-1-transaction-queue"),
		},
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

