// File: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecretKey  string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtKey := os.Getenv("JWT_SECRET_KEY")
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR environment variable not set")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD") // Pode ser vazio

	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		redisDBStr = "0"
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB value: %w", err)
	}

	cfg := &Config{
		JWTSecretKey:  jwtKey,
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
	}

	return cfg, nil
}
