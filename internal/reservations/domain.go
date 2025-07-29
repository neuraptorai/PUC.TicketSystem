// File: internal/reservations/domain.go
package reservations

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Erros de domínio que podem ser usados pelo Service.
var (
	ErrReservationMustHaveItems = errors.New("reservation must contain at least one item")
)

// ReservationStatus define os possíveis estados de uma reserva.
type ReservationStatus string

const (
	StatusAtiva      ReservationStatus = "ATIVA"
	StatusConvertida ReservationStatus = "CONVERTIDA"
	StatusExpirada   ReservationStatus = "EXPIRADA"
	StatusCancelada  ReservationStatus = "CANCELADA"
)

// ReservationItem representa um item dentro de uma reserva.
type ReservationItem struct {
	TicketTypeID string
	Quantity     int
}

// Reservation é a nossa entidade de domínio principal.
type Reservation struct {
	ID        string
	UserID    string
	EventID   string
	Status    ReservationStatus
	ExpiresAt time.Time
	Items     []ReservationItem
	CreatedAt time.Time
}

// NewReservation é uma "factory function" que garante que uma reserva
// seja sempre criada em um estado válido.
func NewReservation(userID, eventID string, items []ReservationItem) (*Reservation, error) {
	if len(items) == 0 {
		return nil, ErrReservationMustHaveItems
	}

	return &Reservation{
		ID:        uuid.NewString(),
		UserID:    userID,
		EventID:   eventID,
		Status:    StatusAtiva,
		ExpiresAt: time.Now().Add(10 * time.Minute), // Regra de negócio: 10 minutos para expirar
		Items:     items,
		CreatedAt: time.Now(),
	}, nil
}
