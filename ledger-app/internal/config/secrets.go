package config

import (
	"log"
)

// Secrets holds all sensitive configuration
type Secrets struct {
	DatabasePassword string
	DatabaseUser     string
	// Add more secrets as needed
}

// LoadSecrets loads secrets from environment variables
// Fails if required secrets are missing
func LoadSecrets() Secrets {
	password := getEnv("COCKROACHDB_PASSWORD", "")
	if password == "" {
		log.Fatal("COCKROACHDB_PASSWORD is required")
	}

	user := getEnv("COCKROACHDB_USER", "root")

	return Secrets{
		DatabasePassword: password,
		DatabaseUser:     user,
	}
}

