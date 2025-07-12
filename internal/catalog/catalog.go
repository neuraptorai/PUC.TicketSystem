// File: internal/catalog/catalog.go
package catalog

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	log *slog.Logger
	rdb *redis.Client
}

func NewHandler(log *slog.Logger, rdb *redis.Client) *Handler {
	return &Handler{log: log, rdb: rdb}
}

// GetAvailability é o handler para GET /events/{eventID}/availability
func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	// Conforme ADR 002, o caminho de consulta lê diretamente do Redis.
	key := fmt.Sprintf("event:%s:stock", eventID)

	// HGETALL retorna todos os campos e valores do hash.
	availability, err := h.rdb.HGetAll(r.Context(), key).Result()
	if err != nil {
		h.log.Error("Failed to get availability from redis", "error", err)
		http.Error(w, "Failed to retrieve availability", http.StatusInternalServerError)
		return
	}

	if len(availability) == 0 {
		http.Error(w, "Event not found or no tickets available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(availability)
}
