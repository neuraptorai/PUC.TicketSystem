// File: internal/platform/web/web.go
package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
)

// ... (interfaces AuthHandler, CatalogHandler, SalesHandler sem alterações) ...
type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	HandleGoogleLogin(w http.ResponseWriter, r *http.Request)
	HandleGoogleCallback(w http.ResponseWriter, r *http.Request)
}
type CatalogHandler interface {
	GetAvailability(w http.ResponseWriter, r *http.Request)
}
type SalesHandler interface {
	CreateReservation(w http.ResponseWriter, r *http.Request)
}

// Adicionamos uma interface para o handler de pagamentos, para o webhook.
type PaymentHandler interface {
	HandleWebhook(w http.ResponseWriter, r *http.Request)
}

func NewRouter(
	secretKey string,
	authHandler AuthHandler,
	catalogHandler CatalogHandler,
	salesHandler SalesHandler,
	paymentHandler PaymentHandler, // Nova dependência
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Get("/health", healthCheckHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login)
		r.Get("/auth/google/login", authHandler.HandleGoogleLogin)
		r.Get("/auth/google/callback", authHandler.HandleGoogleCallback)

		r.Get("/events/{eventID}/availability", catalogHandler.GetAvailability)

		// Webhook de Pagamentos (público, mas a segurança pode ser reforçada)
		r.Post("/payments/webhook", paymentHandler.HandleWebhook)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(secretKey))
			r.Post("/reservations", salesHandler.CreateReservation)
		})
	})

	return r
}

// ... (healthCheckHandler sem alterações) ...
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
