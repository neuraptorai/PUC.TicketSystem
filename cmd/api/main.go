// File: cmd/api/main.go
package main

import (
	// ... outros imports
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
	"github.com/neuraptorai/PUC.TicketSystem/internal/catalog"
	"github.com/neuraptorai/PUC.TicketSystem/internal/config"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/broker" // Import
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/cache"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/web"
	"github.com/neuraptorai/PUC.TicketSystem/internal/sales"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ... logger, config, redis client ...
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("Starting PUC.TicketSystem service...")
	rdb, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("Failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("Successfully connected to Redis")
	seedStock(rdb, logger)

	// 3. Instanciar as dependências

	// Criamos nossa implementação "fake" do broker.
	eventBroker := broker.NewLogBroker(logger)

	userValidator := &auth.MockUserValidator{}
	authHandler := auth.NewHandler(logger, userValidator, cfg.JWTSecretKey)
	catalogHandler := catalog.NewHandler(logger, rdb)

	// Injetamos o broker no handler de sales.
	salesHandler := sales.NewHandler(logger, rdb, eventBroker)

	// 4. Injetar dependências no roteador
	router := web.NewRouter(cfg.JWTSecretKey, authHandler, catalogHandler, salesHandler)

	server := &http.Server{Addr: ":8080", Handler: router}
	// ... (Resto do main sem alterações)
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("Server error", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("Shutdown signal received", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown failed", "error", err)
			server.Close()
		}
		logger.Info("Shutdown complete")
	}
}

// ... seedStock function sem alterações
func seedStock(rdb *redis.Client, log *slog.Logger) {
	ctx := context.Background()
	key := "event:event-123:stock"
	rdb.Del(ctx, key)
	err := rdb.HMSet(ctx, key, map[string]interface{}{
		"ticket-type-standard": 100,
		"ticket-type-vip":      50,
	}).Err()
	if err != nil {
		log.Error("Failed to seed redis stock", "error", err)
	} else {
		log.Info("Redis stock seeded successfully", "event_id", "event-123")
	}
}
