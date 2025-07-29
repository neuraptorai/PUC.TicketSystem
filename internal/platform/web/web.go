// File: internal/platform/web/web.go
package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
	"github.com/neuraptorai/PUC.TicketSystem/internal/catalog"
	"github.com/neuraptorai/PUC.TicketSystem/internal/reservations"
)

func NewRouter(
	secretKey string,
	authHandler *auth.Handler,
	catalogHandler *catalog.Handler,
	reservationHandler *reservations.Handler, // AQUI ESTÁ ELE!
	// paymentHandler *payments.Handler, // Supondo que payments também tenha um Handler
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
		// r.Post("/payments/webhook", paymentHandler.HandleWebhook)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(secretKey))
			r.Post("/reservations", reservationHandler.CreateReservation)
		})
	})

	return r
}

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
