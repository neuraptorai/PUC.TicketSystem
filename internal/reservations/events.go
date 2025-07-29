// File: internal/reservations/events.go (NOVO ARQUIVO)
package reservations

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/events"
)

// Constante para o nome do evento, evitando "magic strings".
const ReservationCreatedEventName = "ReservationCreated"

// ReservationCreatedPayload contém os dados específicos que os consumidores
// precisam saber quando uma reserva é criada.
type ReservationCreatedPayload struct {
	ReservationID string            `json:"reservationId"`
	EventID       string            `json:"eventId"`
	UserID        string            `json:"userId"`
	ExpiresAt     time.Time         `json:"expiresAt"`
	Items         []ReservationItem `json:"items"`
}

// NewReservationCreatedEvent é a nossa "factory function" para criar o evento.
// Ela recebe a entidade de domínio e a transforma no contrato do evento.
func NewReservationCreatedEvent(reservation *Reservation) (*events.EventEnvelope, error) {
	if reservation == nil {
		return nil, fmt.Errorf("reservation cannot be nil")
	}

	payload := ReservationCreatedPayload{
		ReservationID: reservation.ID,
		EventID:       reservation.EventID,
		UserID:        reservation.UserID,
		ExpiresAt:     reservation.ExpiresAt,
		Items:         reservation.Items,
	}

	event := &events.EventEnvelope{
		EventID:      uuid.NewString(),
		EventName:    ReservationCreatedEventName,
		EventVersion: "1.0",
		OccurredAt:   time.Now(),
		Payload:      payload,
	}

	return event, nil
}
