// File: internal/catalog/handler.go (versão final)
package catalog

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// DTO para a resposta da API.
type AvailabilityResponseDTO struct {
	TicketTypeID string `json:"ticketTypeId"`
	Available    int    `json:"available"`
}

// O Handler agora depende do Service.
type Handler struct {
	log     *slog.Logger
	service *Service
}

// O construtor recebe o Service.
func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	// 1. O Handler chama o Service, que contém a lógica de negócio.
	stockItems, err := h.service.GetAvailability(r.Context(), eventID)
	if err != nil {
		// A responsabilidade do Handler é traduzir erros internos para respostas HTTP.
		h.log.Error("service layer returned an error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(stockItems) == 0 {
		http.Error(w, "Event not found or no tickets available", http.StatusNotFound)
		return
	}

	// 2. O Handler transforma o modelo de domínio (StockItem) no DTO da API.
	response := make([]AvailabilityResponseDTO, len(stockItems))
	for i, item := range stockItems {
		response[i] = AvailabilityResponseDTO{
			TicketTypeID: item.TicketTypeID,
			Available:    item.Quantity,
		}
	}

	// 3. O Handler lida com a resposta HTTP.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
