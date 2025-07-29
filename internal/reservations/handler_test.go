// File: internal/reservations/handler_test.go
package reservations

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Definimos uma interface para o nosso serviço para facilitar a criação do mock.
// Embora Go não exija, isso torna a intenção mais clara.
type ReservationCreator interface {
	CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error)
}

// mockReservationService é o nosso dublê para o Service.
type mockReservationService struct {
	CreateReservationFunc func(ctx context.Context, req CreateReservationRequest) (*Reservation, error)
}

func (m *mockReservationService) CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error) {
	return m.CreateReservationFunc(ctx, req)
}

func TestCreateReservationHandler(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	t.Run("Success - 202 Accepted", func(t *testing.T) {
		// Arrange (Organização)
		mockSvc := &mockReservationService{
			CreateReservationFunc: func(ctx context.Context, req CreateReservationRequest) (*Reservation, error) {
				// Simula o serviço retornando uma reserva bem-sucedida.
				res, _ := NewReservation(req.UserID, req.EventID, req.Items)
				return res, nil
			},
		}

		handler := NewHandler(logger, mockSvc)

		// Cria o corpo da requisição JSON
		reqBody := `{"eventId": "event-123", "items": [{"ticketTypeId": "vip", "quantity": 2}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewBufferString(reqBody))

		// Simula o middleware de autenticação adicionando o userID ao contexto
		ctx := context.WithValue(req.Context(), "userID", "user-abc-123")
		req = req.WithContext(ctx)

		// Cria um ResponseRecorder para capturar a resposta
		rr := httptest.NewRecorder()

		// Act (Ação)
		handler.CreateReservation(rr, req)

		// Assert (Verificação)
		if rr.Code != http.StatusAccepted {
			t.Errorf("expected status code %d, but got %d", http.StatusAccepted, rr.Code)
		}

		var respDTO ReservationResponseDTO
		if err := json.Unmarshal(rr.Body.Bytes(), &respDTO); err != nil {
			t.Fatalf("failed to decode response body: %v", err)
		}

		if respDTO.ReservationID == "" {
			t.Error("expected a reservationId in the response, but it was empty")
		}
		if respDTO.Status != "ATIVA" {
			t.Errorf("expected status to be ATIVA, but got %s", respDTO.Status)
		}
	})

	t.Run("Stock Conflict - 409 Conflict", func(t *testing.T) {
		// Arrange
		mockSvc := &mockReservationService{
			CreateReservationFunc: func(ctx context.Context, req CreateReservationRequest) (*Reservation, error) {
				// Simula o serviço retornando um erro de domínio de falta de estoque
				return nil, ErrStockNotAvailable
			},
		}
		handler := NewHandler(logger, mockSvc)
		reqBody := `{"eventId": "event-123", "items": [{"ticketTypeId": "vip", "quantity": 2}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewBufferString(reqBody))
		ctx := context.WithValue(req.Context(), "userID", "user-abc-123")
		rr := httptest.NewRecorder()

		// Act
		handler.CreateReservation(rr, req.WithContext(ctx))

		// Assert
		if rr.Code != http.StatusConflict {
			t.Errorf("expected status code %d, but got %d", http.StatusConflict, rr.Code)
		}
	})

	t.Run("Bad Request - 400 Bad Request on invalid JSON", func(t *testing.T) {
		// Arrange
		mockSvc := &mockReservationService{} // O serviço nem será chamado
		handler := NewHandler(logger, mockSvc)
		reqBody := `{"eventId": "event-123", "items": [` // JSON inválido
		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewBufferString(reqBody))
		ctx := context.WithValue(req.Context(), "userID", "user-abc-123")
		rr := httptest.NewRecorder()

		// Act
		handler.CreateReservation(rr, req.WithContext(ctx))

		// Assert
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, but got %d", http.StatusBadRequest, rr.Code)
		}
	})
}
