// File: internal/reservations/service_test.go
package reservations

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
)

// mockStockRepo é um dublê para o StockRepository.
type mockStockRepo struct {
	ReserveStockFunc   func(ctx context.Context, eventID string, items []ReservationItem) error
	ReleaseStockFunc   func(ctx context.Context, eventID string, items []ReservationItem) error
	releaseStockCalled bool
}

func (m *mockStockRepo) ReserveStock(ctx context.Context, eventID string, items []ReservationItem) error {
	return m.ReserveStockFunc(ctx, eventID, items)
}

func (m *mockStockRepo) ReleaseStock(ctx context.Context, eventID string, items []ReservationItem) error {
	m.releaseStockCalled = true
	return m.ReleaseStockFunc(ctx, eventID, items)
}

// mockReservationRepo é um dublê para o ReservationRepository.
type mockReservationRepo struct {
	CreateFunc func(ctx context.Context, reservation *Reservation) error
}

func (m *mockReservationRepo) Create(ctx context.Context, reservation *Reservation) error {
	return m.CreateFunc(ctx, reservation)
}

// mockEventPublisher é um dublê para o EventPublisher.
type mockEventPublisher struct {
	PublishFunc   func(ctx context.Context, eventData any) error
	publishCalled bool
}

func (m *mockEventPublisher) Publish(ctx context.Context, eventData any) error {
	m.publishCalled = true
	return m.PublishFunc(ctx, eventData)
}

func TestCreateReservation_Success(t *testing.T) {
	// Arrange (Organização)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mockStock := &mockStockRepo{
		ReserveStockFunc: func(ctx context.Context, eventID string, items []ReservationItem) error {
			return nil // Simula sucesso
		},
	}
	mockRepo := &mockReservationRepo{
		CreateFunc: func(ctx context.Context, reservation *Reservation) error {
			return nil // Simula sucesso
		},
	}
	mockPublisher := &mockEventPublisher{
		PublishFunc: func(ctx context.Context, eventData any) error {
			return nil // Simula sucesso
		},
	}

	service := NewService(logger, mockStock, mockRepo, mockPublisher)

	req := CreateReservationRequest{
		UserID:  "user-123",
		EventID: "event-456",
		Items:   []ReservationItem{{TicketTypeID: "vip", Quantity: 2}},
	}

	// Act (Ação)
	reservation, err := service.CreateReservation(context.Background(), req)

	// Assert (Verificação)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if reservation == nil {
		t.Fatal("expected reservation to be created, but it was nil")
	}
	if reservation.Status != StatusAtiva {
		t.Errorf("expected status to be ATIVA, but got %s", reservation.Status)
	}
	if !mockPublisher.publishCalled {
		t.Error("expected EventPublisher.Publish to be called, but it wasn't")
	}
}

func TestCreateReservation_StockNotAvailable(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mockStock := &mockStockRepo{
		ReserveStockFunc: func(ctx context.Context, eventID string, items []ReservationItem) error {
			return ErrStockNotAvailable // Simula falha de estoque
		},
	}
	// Para este teste, os outros mocks nem precisam ser instanciados,
	// mas vamos mantê-los por clareza.
	mockRepo := &mockReservationRepo{}
	mockPublisher := &mockEventPublisher{}

	service := NewService(logger, mockStock, mockRepo, mockPublisher)
	req := CreateReservationRequest{UserID: "user-123", EventID: "event-456", Items: []ReservationItem{{TicketTypeID: "vip", Quantity: 2}}}

	// Act
	_, err := service.CreateReservation(context.Background(), req)

	// Assert
	if !errors.Is(err, ErrStockNotAvailable) {
		t.Errorf("expected error to be ErrStockNotAvailable, but got %v", err)
	}
	if mockPublisher.publishCalled {
		t.Error("expected EventPublisher.Publish NOT to be called, but it was")
	}
}

func TestCreateReservation_PersistenceFails_StockIsReleased(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mockStock := &mockStockRepo{
		ReserveStockFunc: func(ctx context.Context, eventID string, items []ReservationItem) error {
			return nil // Reserva no Redis funciona
		},
		ReleaseStockFunc: func(ctx context.Context, eventID string, items []ReservationItem) error {
			return nil // Devolução também funciona
		},
	}
	mockRepo := &mockReservationRepo{
		CreateFunc: func(ctx context.Context, reservation *Reservation) error {
			return errors.New("database connection lost") // Simula falha no BD
		},
	}
	mockPublisher := &mockEventPublisher{}

	service := NewService(logger, mockStock, mockRepo, mockPublisher)
	req := CreateReservationRequest{UserID: "user-123", EventID: "event-456", Items: []ReservationItem{{TicketTypeID: "vip", Quantity: 2}}}

	// Act
	_, err := service.CreateReservation(context.Background(), req)

	// Assert
	if err == nil {
		t.Error("expected an error, but got nil")
	}
	// A VERIFICAÇÃO MAIS IMPORTANTE:
	if !mockStock.releaseStockCalled {
		t.Error("expected StockRepository.ReleaseStock to be called for compensation, but it wasn't")
	}
}
