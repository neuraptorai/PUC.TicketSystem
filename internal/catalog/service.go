// File: internal/catalog/service.go
package catalog

import (
	"context"
	"fmt"
	"log/slog"
)

// Service encapsula a lógica de negócio do catálogo.
// Suas dependências são sempre interfaces (ports).
type Service struct {
	log  *slog.Logger
	repo AvailabilityRepository
}

// NewService é o construtor do nosso serviço.
func NewService(log *slog.Logger, repo AvailabilityRepository) *Service {
	return &Service{log: log, repo: repo}
}

// GetAvailability é o nosso caso de uso. Ele orquestra a busca por disponibilidade.
func (s *Service) GetAvailability(ctx context.Context, eventID string) ([]StockItem, error) {
	s.log.Info("fetching availability for event", "eventID", eventID)

	stockItems, err := s.repo.GetStockByEventID(ctx, eventID)
	if err != nil {
		// No futuro, poderíamos mapear erros do repositório para erros de domínio mais específicos.
		return nil, fmt.Errorf("service failed to get availability: %w", err)
	}

	// NOTA: Se houvesse lógica de negócio complexa, ela estaria aqui.
	// Por exemplo, filtrar resultados, combinar com outros dados, etc.
	// Por enquanto, o service atua como um orquestrador direto.

	if len(stockItems) == 0 {
		// Poderíamos definir um erro de domínio padrão aqui.
		// return nil, ErrEventNotFound
	}

	return stockItems, nil
}
