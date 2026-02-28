package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	ClashHost string
	AuthToken string
}

// Default returns a Config with default values.
func Default() Config {
	return Config{
		ClashHost: "http://127.0.0.1:9090",
		AuthToken: "",
	}
}

// LoadFromEnv loads configuration from environment variables,
// falling back to defaults for any unset values.
func LoadFromEnv() Config {
	cfg := Default()

	if host := os.Getenv("CLASH_HOST"); host != "" {
		cfg.ClashHost = host
	}
	if token := os.Getenv("CLASH_TOKEN"); token != "" {
		cfg.AuthToken = token
	}

	return cfg
}

// Validate checks that the configuration is usable.
func (c Config) Validate() error {
	if c.ClashHost == "" {
		return fmt.Errorf("clash host must not be empty")
	}
	return nil
}
