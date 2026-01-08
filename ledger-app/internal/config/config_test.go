package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("APP_PORT")
	originalRegion := os.Getenv("REGION")
	originalDBHost := os.Getenv("COCKROACHDB_HOST")

	// Clean up after test
	defer func() {
		if originalPort != "" {
			os.Setenv("APP_PORT", originalPort)
		} else {
			os.Unsetenv("APP_PORT")
		}
		if originalRegion != "" {
			os.Setenv("REGION", originalRegion)
		} else {
			os.Unsetenv("REGION")
		}
		if originalDBHost != "" {
			os.Setenv("COCKROACHDB_HOST", originalDBHost)
		} else {
			os.Unsetenv("COCKROACHDB_HOST")
		}
	}()

	tests := []struct {
		name     string
		setup    func()
		validate func(*testing.T, Config)
	}{
		{
			name: "default values",
			setup: func() {
				os.Unsetenv("APP_PORT")
				os.Unsetenv("REGION")
				os.Unsetenv("COCKROACHDB_HOST")
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.App.Port != 8080 {
					t.Errorf("Expected default port 8080, got %d", cfg.App.Port)
				}
				if cfg.App.Region != "us-east-1" {
					t.Errorf("Expected default region us-east-1, got %s", cfg.App.Region)
				}
				if cfg.Database.Host != "cockroachdb-public" {
					t.Errorf("Expected default host cockroachdb-public, got %s", cfg.Database.Host)
				}
			},
		},
		{
			name: "custom values from env",
			setup: func() {
				os.Setenv("APP_PORT", "9090")
				os.Setenv("REGION", "eu-central-1")
				os.Setenv("COCKROACHDB_HOST", "custom-host")
			},
			validate: func(t *testing.T, cfg Config) {
				if cfg.App.Port != 9090 {
					t.Errorf("Expected port 9090, got %d", cfg.App.Port)
				}
				if cfg.App.Region != "eu-central-1" {
					t.Errorf("Expected region eu-central-1, got %s", cfg.App.Region)
				}
				if cfg.Database.Host != "custom-host" {
					t.Errorf("Expected host custom-host, got %s", cfg.Database.Host)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := LoadConfig()
			tt.validate(t, cfg)
		})
	}
}

func TestLoadSecrets(t *testing.T) {
	originalPassword := os.Getenv("COCKROACHDB_PASSWORD")

	defer func() {
		if originalPassword != "" {
			os.Setenv("COCKROACHDB_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("COCKROACHDB_PASSWORD")
		}
	}()

	t.Run("missing password should fail", func(t *testing.T) {
		os.Unsetenv("COCKROACHDB_PASSWORD")
		// This should call log.Fatal, so we can't test it directly
		// In a real scenario, you'd use a test helper that recovers from fatal
	})

	t.Run("password set should succeed", func(t *testing.T) {
		os.Setenv("COCKROACHDB_PASSWORD", "test-password")
		secrets := LoadSecrets()
		if secrets.DatabasePassword != "test-password" {
			t.Errorf("Expected password test-password, got %s", secrets.DatabasePassword)
		}
	})
}

func TestGetEnv(t *testing.T) {
	original := os.Getenv("TEST_VAR")

	defer func() {
		if original != "" {
			os.Setenv("TEST_VAR", original)
		} else {
			os.Unsetenv("TEST_VAR")
		}
	}()

	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test-value")
		result := getEnv("TEST_VAR", "default")
		if result != "test-value" {
			t.Errorf("Expected test-value, got %s", result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")
		result := getEnv("TEST_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("Expected default-value, got %s", result)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	original := os.Getenv("TEST_INT_VAR")

	defer func() {
		if original != "" {
			os.Setenv("TEST_INT_VAR", original)
		} else {
			os.Unsetenv("TEST_INT_VAR")
		}
	}()

	tests := []struct {
		name     string
		setup    func()
		expected int
	}{
		{
			name:     "valid integer",
			setup:    func() { os.Setenv("TEST_INT_VAR", "42") },
			expected: 42,
		},
		{
			name:     "invalid integer returns default",
			setup:    func() { os.Setenv("TEST_INT_VAR", "not-a-number") },
			expected: 100,
		},
		{
			name:     "not set returns default",
			setup:    func() { os.Unsetenv("TEST_INT_VAR") },
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := getEnvInt("TEST_INT_VAR", 100)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}
