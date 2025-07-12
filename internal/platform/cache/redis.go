// File: internal/platform/cache/redis.go
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient cria e retorna um novo cliente Redis.
// Ele também executa um PING para garantir que a conexão está ativa.
func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Verifica a conexão.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return rdb, nil
}
