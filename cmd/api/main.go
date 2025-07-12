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

	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/web"
)

func main() {
	// 1. Inicialização do Logger Estruturado (log/slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("Starting PUC.TicketSystem service...")

	// 2. Inicialização do Servidor Web com o roteador Chi
	// A função NewRouter agora retorna um roteador chi configurado.
	router := web.NewRouter()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      router, // O roteador do Chi é um http.Handler
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 3. Goroutine para o Servidor
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// 4. Graceful Shutdown
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
