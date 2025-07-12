// File: internal/platform/web/web.go
package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter cria um novo roteador chi e registra os handlers da aplicação.
func NewRouter() http.Handler {
	// Instanciamos o roteador do Chi.
	r := chi.NewRouter()

	// Adicionamos middlewares que serão úteis.
	// O middleware.RequestID adiciona um ID único a cada requisição.
	// O middleware.Logger é um logger de requisições básico.
	// O middleware.Recoverer previne que a aplicação pare em caso de panic.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Pode ser substituído por um middleware slog customizado no futuro.
	r.Use(middleware.Recoverer)

	// Registramos nosso endpoint de health check usando o método GET.
	r.Get("/health", healthCheckHandler)

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

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
