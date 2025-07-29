// File: internal/reservations/ports.go
package reservations

import "context"

// StockRepository define a porta para operações no estoque (implementado pelo Redis).
type StockRepository interface {
	ReserveStock(ctx context.Context, eventID string, items []ReservationItem) error
	ReleaseStock(ctx context.Context, eventID string, items []ReservationItem) error
}

// ReservationRepository define a porta para persistência da entidade Reserva (implementado pelo PostgreSQL).
type ReservationRepository interface {
	Create(ctx context.Context, reservation *Reservation) error
	// ... outros métodos como GetByID, UpdateStatus, etc.
}

// EventPublisher define a porta para publicação de eventos (implementado pelo RabbitMQ/NATS).
type EventPublisher interface {
	Publish(ctx context.Context, eventData any) error
}
