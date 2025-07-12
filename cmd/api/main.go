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

	"github.com/neuraptorai/PUC.TicketSystem/config"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. Carregar a configuração
	// Este é um dos primeiros passos, pois a configuração é necessária para
	// inicializar outros componentes.
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting PUC.TicketSystem service...")

	// 2. Instanciar as dependências (Composition Root)
	userValidator := &auth.MockUserValidator{}

	// Injetamos a chave secreta do JWT (carregada do config) no handler.
	authHandler := auth.NewHandler(logger, userValidator, cfg.JWTSecretKey)

	router := web.NewRouter(authHandler)

	// 3. O restante da configuração do servidor permanece o mesmo.
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

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
