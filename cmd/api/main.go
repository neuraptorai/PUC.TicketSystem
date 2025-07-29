// File: cmd/api/main.go
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neuraptorai/PUC.TicketSystem/internal/adapters/postgresrepo"
	"github.com/neuraptorai/PUC.TicketSystem/internal/adapters/rabbitmq"
	"github.com/neuraptorai/PUC.TicketSystem/internal/adapters/redisrepo"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
	"github.com/neuraptorai/PUC.TicketSystem/internal/catalog"
	"github.com/neuraptorai/PUC.TicketSystem/internal/config"
	"github.com/neuraptorai/PUC.TicketSystem/internal/database"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/cache"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/queue"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/web"
	"github.com/neuraptorai/PUC.TicketSystem/internal/reservations"
	"github.com/neuraptorai/PUC.TicketSystem/internal/user"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("Failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("Successfully connected to Redis")
	seedStock(redisClient, logger)

	rabbitConn, err := queue.NewRabbitMQConnection(cfg.RABBITMQ_DSN)
	if err != nil {
		log.Fatal("failed to connect to rabbitmq", "error", err)
	}
	defer rabbitConn.Close()
	const domainExchange = "domain_events"
	if err := rabbitConn.SetupDomainExchanges(domainExchange); err != nil {
		log.Fatal("failed to setup rabbitmq exchanges", "error", err)
	}
	logger.Info("Successfully connected to RabbitMQ")

	// == Event Publisher ==
	eventPublisher := rabbitmq.NewEventPublisher(rabbitConn.Channel, logger, domainExchange)

	// == Dependências Repositórios ==
	availabilityRepo := redisrepo.NewAvailabilityRepo(redisClient, logger)
	userRepo := user.NewRepository(logger, dbPool)
	reservationRepo := postgresrepo.NewReservationRepo(dbPool, logger)
	stockRepo := redisrepo.NewStockRepo(redisClient, logger)

	// == Services ==
	catalogSvc := catalog.NewService(logger, availabilityRepo)
	reservationSvc := reservations.NewService(logger, stockRepo, reservationRepo, eventPublisher)

	// --- Instanciação dos Componentes ---
	authHandler := auth.NewHandler(logger, userRepo, cfg.JWTSecretKey, cfg.GoogleOAuthConfig)
	catalogHandler := catalog.NewHandler(logger, catalogSvc)
	reservationHandler := reservations.NewHandler(logger, reservationSvc)
	// O paymentHandler agora também precisa do broker para publicar o evento interno.

	// --- Configuração do Roteador ---
	router := web.NewRouter(cfg.JWTSecretKey, authHandler, catalogHandler, reservationHandler)

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
