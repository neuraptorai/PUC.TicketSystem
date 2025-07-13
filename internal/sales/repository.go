// File: internal/sales/repository.go
package sales

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository encapsula o acesso ao banco de dados para o contexto de vendas.
type Repository struct {
	log *slog.Logger
	db  *pgxpool.Pool
}

// NewRepository cria uma nova instância do repositório.
func NewRepository(log *slog.Logger, db *pgxpool.Pool) *Repository {
	return &Repository{log: log, db: db}
}

// CreateReservation persiste uma nova reserva no banco de dados.
func (r *Repository) CreateReservation(ctx context.Context, data ReservationData) error {
	query := `
		INSERT INTO reservations (id, user_id, event_id, ticket_type_id, quantity, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	// Define um tempo de expiração para a reserva (ex: 10 minutos).
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	status := "RESERVED"

	reservationUUID, err := uuid.Parse(data.ReservationID)
	if err != nil {
		return err
	}
	userUUID, err := uuid.Parse(data.UserID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query,
		reservationUUID,
		userUUID,
		data.EventID,
		data.TicketTypeID,
		data.Quantity,
		status,
		time.Now().UTC(),
		expiresAt,
	)

	if err != nil {
		r.log.ErrorContext(ctx, "failed to insert reservation", "error", err)
	}

	return err
}
