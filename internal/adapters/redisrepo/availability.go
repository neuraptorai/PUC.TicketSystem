// File: internal/adapters/redisrepo/availability.go
package redisrepo

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/neuraptorai/PUC.TicketSystem/internal/catalog"
	"github.com/redis/go-redis/v9"
)

// AvailabilityRepo é nosso Adapter. Ele "traduz" as chamadas
// da interface para comandos específicos do Redis.
type AvailabilityRepo struct {
	rdb *redis.Client
	log *slog.Logger
}

// NewAvailabilityRepo é o construtor do nosso adapter.
func NewAvailabilityRepo(rdb *redis.Client, log *slog.Logger) *AvailabilityRepo {
	return &AvailabilityRepo{rdb: rdb, log: log}
}

// GetStockByEventID implementa o contrato da interface.
func (r *AvailabilityRepo) GetStockByEventID(ctx context.Context, eventID string) ([]catalog.StockItem, error) {
	// O detalhe do formato da chave vive e morre aqui, encapsulado.
	key := fmt.Sprintf("event:%s:stock", eventID)

	// 1. Chamar o Redis (o detalhe de infraestrutura).
	rawStock, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stock from redis for event %s: %w", eventID, err)
	}

	// 2. TRADUZIR a resposta do Redis (map[string]string) para o nosso modelo de domínio ([]StockItem).
	// Esta tradução é a principal responsabilidade do Adapter.
	stockItems := make([]catalog.StockItem, 0, len(rawStock))
	for ticketTypeID, quantityStr := range rawStock {
		quantity, err := strconv.Atoi(quantityStr)
		if err != nil {
			r.log.Warn("failed to parse stock quantity from redis", "eventID", eventID, "ticketTypeID", ticketTypeID, "value", quantityStr)
			continue // Pula itens malformados no cache.
		}

		stockItems = append(stockItems, catalog.StockItem{
			TicketTypeID: ticketTypeID,
			Quantity:     quantity,
		})
	}

	return stockItems, nil
}
