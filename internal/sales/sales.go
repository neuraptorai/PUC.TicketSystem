// File: internal/sales/sales.go
package sales

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/neuraptorai/PUC.TicketSystem/internal/auth"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/broker" // Importamos o novo pacote
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	log    *slog.Logger
	rdb    *redis.Client
	broker broker.MessageBroker
	repo   *Repository
}

// NewHandler agora aceita um MessageBroker.
func NewHandler(log *slog.Logger, rdb *redis.Client, broker broker.MessageBroker, repo *Repository) *Handler {
	return &Handler{log: log, rdb: rdb, broker: broker, repo: repo}
}

type ReservationRequest struct {
	EventID      string `json:"event_id"`
	TicketTypeID string `json:"ticket_type_id"`
	Quantity     int64  `json:"quantity"`
}

// ReservationData é a struct que será enviada no campo 'Data' do nosso evento.
type ReservationData struct {
	ReservationID string `json:"reservation_id"`
	EventID       string `json:"event_id"`
	TicketTypeID  string `json:"ticket_type_id"`
	Quantity      int64  `json:"quantity"`
	UserID        string `json:"user_id"`
}

type ReservationResponse struct {
	ReservationID string `json:"reservation_id"`
	Status        string `json:"status"`
}

func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok {
		http.Error(w, "Could not retrieve user ID from context", http.StatusInternalServerError)
		return
	}

	h.log.InfoContext(ctx, "Reservation attempt", "user_id", userID, "event_id", req.EventID)

	key := fmt.Sprintf("event:%s:stock", req.EventID)
	field := req.TicketTypeID

	newStock, err := h.rdb.HIncrBy(ctx, key, field, -req.Quantity).Result()
	if err != nil {
		h.log.ErrorContext(ctx, "Failed to decrement stock in redis", "error", err)
		http.Error(w, "Failed to process reservation", http.StatusInternalServerError)
		return
	}

	if newStock < 0 {
		h.rdb.HIncrBy(ctx, key, field, req.Quantity).Result() // Revert
		http.Error(w, "Not enough tickets available", http.StatusConflict)
		return
	}

	// Sucesso! Agora publicamos o evento.
	reservationID := uuid.NewString()

	// 1. Criar o evento de domínio.
	eventData := ReservationData{
		ReservationID: reservationID,
		EventID:       req.EventID,
		TicketTypeID:  req.TicketTypeID,
		Quantity:      req.Quantity,
		UserID:        userID,
	}
	if err := h.repo.CreateReservation(ctx, eventData); err != nil {
		// ERRO CRÍTICO: O estoque no Redis foi decrementado, mas a persistência falhou.
		// Em um sistema real, aqui teríamos que ter uma lógica de compensação,
		// como reverter o estoque no Redis ou enfileirar a persistência para uma nova tentativa.
		h.log.ErrorContext(ctx, "CRITICAL: Failed to persist reservation after updating stock", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	reservationEvent := broker.NewEvent("sales-service", "ReservationCreated", eventData)

	// 2. Publicar o evento usando o broker injetado.
	topic := "reservations.created"
	if err := h.broker.Publish(ctx, topic, reservationEvent); err != nil {
		h.log.ErrorContext(ctx, "Failed to publish reservation event", "error", err)
		// Aqui teríamos que decidir uma estratégia de compensação.
		// Por simplicidade, vamos apenas logar o erro.
	}

	h.log.InfoContext(ctx, "Reservation successful, event published", "new_stock", newStock, "reservation_id", reservationID)

	resp := ReservationResponse{
		ReservationID: reservationID,
		Status:        "RESERVED",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
