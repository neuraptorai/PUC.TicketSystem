// File: internal/platform/web/web.go
package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
)

type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
}
type CatalogHandler interface {
	GetAvailability(w http.ResponseWriter, r *http.Request)
}
type SalesHandler interface {
	CreateReservation(w http.ResponseWriter, r *http.Request)
}

func NewRouter(
	secretKey string,
	authHandler AuthHandler,
	catalogHandler CatalogHandler,
	salesHandler SalesHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", healthCheckHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)

		// Rotas do Catálogo (Consulta - Públicas)
		r.Get("/events/{eventID}/availability", catalogHandler.GetAvailability)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(secretKey))
			r.Post("/reservations", salesHandler.CreateReservation)
		})
	})

	return r
}

// ... healthCheckHandler sem alterações
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
