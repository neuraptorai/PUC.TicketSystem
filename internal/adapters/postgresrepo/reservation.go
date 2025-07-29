// File: internal/adapters/postgresrepo/reservation.go
package postgresrepo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neuraptorai/PUC.TicketSystem/internal/reservations"
)

// ReservationRepo é a nossa implementação concreta do repositório para PostgreSQL.
type ReservationRepo struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewReservationRepo é o construtor do nosso adapter.
func NewReservationRepo(pool *pgxpool.Pool, log *slog.Logger) *ReservationRepo {
	return &ReservationRepo{pool: pool, log: log}
}

func (r *ReservationRepo) Create(ctx context.Context, reservation *reservations.Reservation) error {
	// 1. Inicia a transação a partir do pool de conexões.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// `defer tx.Rollback` garante que, se qualquer erro ocorrer, a transação será desfeita.
	// O `Rollback` é ignorado se `Commit` for chamado com sucesso.
	defer tx.Rollback(ctx)

	// 2. Insere o registro principal na tabela `reservations`.
	sqlReservation := `
        INSERT INTO reservations (id, user_id, event_id, status, expires_at, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err = tx.Exec(ctx, sqlReservation,
		reservation.ID,
		reservation.UserID,
		reservation.EventID,
		reservation.Status,
		reservation.ExpiresAt,
		reservation.CreatedAt,
	)
	if err != nil {
		r.log.Error("failed to insert reservation", "error", err)
		return fmt.Errorf("failed to execute reservation insert: %w", err)
	}

	// 3. Itera sobre os itens e insere cada um na tabela `reservation_items`.
	sqlItem := `
        INSERT INTO reservation_items (reservation_id, ticket_type_id, quantity)
        VALUES ($1, $2, $3)
    `
	for _, item := range reservation.Items {
		_, err = tx.Exec(ctx, sqlItem,
			reservation.ID,
			item.TicketTypeID,
			item.Quantity,
		)
		if err != nil {
			r.log.Error("failed to insert reservation item", "reservation_id", reservation.ID, "error", err)
			return fmt.Errorf("failed to execute reservation item insert: %w", err)
		}
	}

	// 4. Se todas as inserções foram bem-sucedidas, confirma a transação.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
