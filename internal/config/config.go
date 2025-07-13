// File: internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	// ... (outras configs)
	JWTSecretKey      string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	DatabaseDSN       string
	GoogleOAuthConfig *oauth2.Config
	RABBITMQ_DSN      string // DSN para conexão com RabbitMQ
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	// ... (carregamento de outras vars)
	jwtKey := os.Getenv("JWT_SECRET_KEY")
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT_SECRET_KEY environment variable not set")
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR environment variable not set")
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		redisDBStr = "0"
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB value: %w", err)
	}

	// Configuração do OAuth do Google
	googleOAuthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	// Monta a DSN do PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// Verifica se a DSN do RabbitMQ está definida
	rabbitMQDSN := os.Getenv("RABBITMQ_DSN")
	if rabbitMQDSN == "" {
		return nil, fmt.Errorf("RABBITMQ_DSN environment variable not set")
	}

	cfg := &Config{
		JWTSecretKey:      jwtKey,
		RedisAddr:         redisAddr,
		RedisPassword:     redisPassword,
		RedisDB:           redisDB,
		DatabaseDSN:       dsn,
		GoogleOAuthConfig: googleOAuthConfig,
		RABBITMQ_DSN:      rabbitMQDSN,
	}

	return cfg, nil
}
