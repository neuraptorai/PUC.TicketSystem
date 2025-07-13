// File: internal/payments/repository.go
package payments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	log *slog.Logger
	db  *pgxpool.Pool
}

func NewRepository(log *slog.Logger, db *pgxpool.Pool) *Repository {
	return &Repository{log: log, db: db}
}

// FinalizeOrder executa a transação para converter uma reserva em um pedido.
func (r *Repository) FinalizeOrder(ctx context.Context, payload WebhookPayload) error {
	// PASSO DE IDEMPOTÊNCIA: Verifica se já existe um pedido para esta reserva.
	var orderID string
	checkQuery := `SELECT id FROM orders WHERE reservation_id = $1`
	reservationUUID, err := uuid.Parse(payload.ReservationID)
	if err != nil {
		return fmt.Errorf("invalid reservation_id format: %w", err)
	}

	err = r.db.QueryRow(ctx, checkQuery, reservationUUID).Scan(&orderID)
	if err == nil && orderID != "" {
		// Pedido já existe. Retornamos sucesso sem fazer nada.
		r.log.WarnContext(ctx, "Order already processed for this reservation. Skipping.", "reservation_id", payload.ReservationID, "existing_order_id", orderID)
		return nil
	}
	// Se o erro não for 'no rows', algo inesperado aconteceu.
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check for existing order: %w", err)
	}

	// Inicia a transação com o banco de dados.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Passo 1: Atualiza o status da reserva para 'CONVERTED'.
	updateReservationQuery := `UPDATE reservations SET status = 'CONVERTED' WHERE id = $1`

	cmdTag, err := tx.Exec(ctx, updateReservationQuery, reservationUUID)
	if err != nil {
		return fmt.Errorf("failed to update reservation status: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("reservation not found or already processed: %s", payload.ReservationID)
	}

	// Passo 2: Cria o registro na tabela 'orders'.
	createOrderQuery := `
		INSERT INTO orders (user_id, reservation_id, status, total_amount)
		SELECT user_id, id, 'COMPLETED', $2 FROM reservations WHERE id = $1
	`
	_, err = tx.Exec(ctx, createOrderQuery, reservationUUID, payload.Amount)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Se tudo correu bem, efetiva a transação.
	return tx.Commit(ctx)
}
