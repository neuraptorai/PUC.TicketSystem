// File: internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecretKey string
}

func Load() (*Config, error) {

	_ = godotenv.Load()

	jwtKey := os.Getenv("JWT_SECRET_KEY")
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}

	cfg := &Config{
		JWTSecretKey: jwtKey,
	}

	return cfg, nil
}
