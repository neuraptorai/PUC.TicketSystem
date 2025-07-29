// File: internal/catalog/repository.go
package catalog

import (
	"context"
)

// StockItem é uma representação do nosso domínio para um item em estoque.
// Note que não é um DTO de API e não tem tags JSON.
type StockItem struct {
	TicketTypeID string
	Quantity     int
}

// AvailabilityRepository é a nossa porta. Define o que nosso domínio
// precisa, sem se preocupar com a implementação (Redis, SQL, etc.).
type AvailabilityRepository interface {
	GetStockByEventID(ctx context.Context, eventID string) ([]StockItem, error)
}
