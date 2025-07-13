// File: cmd/api/main.go
package main

import (
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
	"github.com/neuraptorai/PUC.TicketSystem/internal/database"
	"github.com/neuraptorai/PUC.TicketSystem/internal/payments"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/broker"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/cache"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/web"
	"github.com/neuraptorai/PUC.TicketSystem/internal/sales"
	"github.com/neuraptorai/PUC.TicketSystem/internal/user"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// ... (config, redis, seedStock) ...
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	if err := database.RunMigrations(cfg.DatabaseDSN, logger); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}
	dbPool, err := database.NewPostgresPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("Failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("Successfully connected to PostgreSQL")
	rdb, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("Failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("Successfully connected to Redis")
	seedStock(rdb, logger)

	// Conectamos ao RabbitMQ
	rabbitBroker, err := broker.NewRabbitMQBroker(cfg.RABBITMQ_DSN, logger)
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitBroker.Close()
	logger.Info("Successfully connected to RabbitMQ")

	// == Dependências Repositórios ==
	salesRepo := sales.NewRepository(logger, dbPool)
	paymentsRepo := payments.NewRepository(logger, dbPool)

	// --- Instanciação dos Componentes ---
	userRepo := user.NewRepository(logger, dbPool)
	authHandler := auth.NewHandler(logger, userRepo, cfg.JWTSecretKey, cfg.GoogleOAuthConfig)
	catalogHandler := catalog.NewHandler(logger, rdb)
	salesHandler := sales.NewHandler(logger, rdb, rabbitBroker, salesRepo)
	paymentGateway := payments.NewFakePaymentGateway(logger)
	// O paymentHandler agora também precisa do broker para publicar o evento interno.
	paymentHandler := payments.NewEventHandler(logger, paymentGateway, rabbitBroker, paymentsRepo)

	// --- Inscrição de Eventos ---
	go rabbitBroker.StartConsumer(
		"reservations.created",               // Exchange
		"payments.reservation_created.queue", // Queue
		"ReservationCreated",                 // Routing Key
		paymentHandler.HandleReservationCreated,
	)

	go rabbitBroker.StartConsumer(
		"payments.processed",
		"orders.payment_processed.queue",
		"PaymentProcessed",
		paymentHandler.HandlePaymentProcessed,
	)

	// --- Configuração do Roteador ---
	router := web.NewRouter(cfg.JWTSecretKey, authHandler, catalogHandler, salesHandler, paymentHandler)

	// ... (Resto do main sem alterações) ...
	server := &http.Server{Addr: ":8080", Handler: router}
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
