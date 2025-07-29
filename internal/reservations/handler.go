// File: internal/reservations/handler.go
package reservations

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Handler struct {
	log     *slog.Logger
	service ServiceInterface
}

func NewHandler(log *slog.Logger, service ServiceInterface) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	// Obter userID do contexto JWT (preenchido por um middleware).
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decodificar o corpo da requisição para um DTO.
	var reqDTO CreateReservationRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mapear DTO para a requisição do serviço.
	serviceReq := CreateReservationRequest{
		EventID: reqDTO.EventID,
		UserID:  userID,
		Items:   reqDTO.ToDomainItems(),
	}

	// Chamar o serviço.
	reservation, err := (h.service).CreateReservation(r.Context(), serviceReq)
	if err != nil {
		// Mapear erros de domínio para respostas HTTP.
		if errors.Is(err, ErrStockNotAvailable) {
			http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
			return
		}
		// ... outros mapeamentos de erro ...
		h.log.Error("failed to create reservation", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Responder com 202 Accepted e o DTO de resposta.
	responseDTO := ToResponseDTO(reservation)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(responseDTO)
}
