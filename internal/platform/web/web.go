// File: internal/platform/web/web.go
package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// AuthHandler define a interface que esperamos para os handlers de autenticação.
// Isso evita que o pacote `web` precise importar o pacote `auth`,
// mantendo as dependências na direção correta (web não conhece auth).
type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
}

// NewRouter cria o roteador principal e registra todas as rotas.
func NewRouter(authHandler AuthHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Endpoint público de health check.
	r.Get("/health", healthCheckHandler)

	// Agrupamos as rotas da API sob um prefixo versionado /api/v1.
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authHandler.Login) //

		// Rotas protegidas (exemplo futuro)
		// r.Group(func(r chi.Router) {
		//  r.Use(AuthMiddleware(jwtSecretKey)) // Middleware de validação JWT
		//  r.Get("/me", protectedHandler)
		// })
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
