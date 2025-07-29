// File: internal/reservations/service.go
package reservations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Erros de domínio específicos do serviço para serem retornados ao Handler.
var (
	ErrStockNotAvailable = errors.New("stock not available for one or more items")
	// ... outros erros como ErrUserHasActiveReservation ...
)

type ServiceInterface interface {
	CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error)
}

var _ ServiceInterface = (*Service)(nil)

type Service struct {
	log          *slog.Logger
	stockRepo    StockRepository
	reservations ReservationRepository
	publisher    EventPublisher
}

// NewService é o construtor que injeta as dependências (interfaces).
func NewService(log *slog.Logger, stockRepo StockRepository, reservationsRepo ReservationRepository, publisher EventPublisher) *Service {
	return &Service{
		log:          log,
		stockRepo:    stockRepo,
		reservations: reservationsRepo,
		publisher:    publisher,
	}
}

// CreateReservationRequest é o DTO de entrada para o serviço.
type CreateReservationRequest struct {
	EventID string
	UserID  string
	Items   []ReservationItem
}

func (s *Service) CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error) {
	log := s.log.With("userID", req.UserID, "eventID", req.EventID)
	log.Info("starting reservation process")

	// PASSO 1: Tenta reservar o estoque no Redis. Esta é a operação crítica e primeira.
	if err := s.stockRepo.ReserveStock(ctx, req.EventID, req.Items); err != nil {
		log.Warn("failed to reserve stock", "error", err)
		// Assume-se que o adapter do repo retorna este erro específico.
		return nil, ErrStockNotAvailable
	}

	// PASSO 2: Se o estoque foi reservado, cria a entidade de domínio.
	reservation, err := NewReservation(req.UserID, req.EventID, req.Items)
	if err != nil {
		// LÓGICA DE COMPENSAÇÃO (Saga): Se a criação da entidade falhar, devemos devolver o estoque.
		log.Error("failed to create reservation entity, releasing stock", "error", err)
		s.stockRepo.ReleaseStock(context.Background(), req.EventID, req.Items)
		return nil, fmt.Errorf("failed to create reservation entity: %w", err)
	}

	// PASSO 3: Persiste a reserva no banco de dados (PostgreSQL).
	if err := s.reservations.Create(ctx, reservation); err != nil {
		// LÓGICA DE COMPENSAÇÃO (Saga): Se a persistência no BD falhar, também devolvemos o estoque.
		log.Error("failed to persist reservation, releasing stock", "reservationID", reservation.ID, "error", err)
		s.stockRepo.ReleaseStock(context.Background(), req.EventID, req.Items)
		return nil, fmt.Errorf("could not save reservation: %w", err)
	}

	// PASSO 4: Publique o evento de sucesso para ser consumido pelo módulo de pagamentos.
	event, err := NewReservationCreatedEvent(reservation) // AGORA ESTA FUNÇÃO EXISTE!
	if err != nil {
		// Este erro seria grave, indicando um bug interno.
		log.Error("CRITICAL: failed to create reservation event object", "reservationID", reservation.ID, "error", err)
		// A reserva já foi salva, então não retornamos erro ao usuário, mas o alerta é fundamental.
		// A lógica de compensação aqui seria mais complexa (ex: cancelar a reserva).
	} else {
		if err := s.publisher.Publish(ctx, event); err != nil {
			// ALERTA CRÍTICO: A reserva foi salva, mas o evento não foi publicado.
			log.Error("CRITICAL: reservation created but event publication failed", "reservationID", reservation.ID, "error", err)
		}
	}

	log.Info("reservation successful", "reservationID", reservation.ID)
	return reservation, nil
}
