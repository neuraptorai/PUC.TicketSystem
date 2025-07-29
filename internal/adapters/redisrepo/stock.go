// File: internal/adapters/redisrepo/stock.go
package redisrepo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/neuraptorai/PUC.TicketSystem/internal/reservations"
	"github.com/redis/go-redis/v9"
	// Importa o pacote com a interface e a entidade
)

// Erro específico retornado pelo repositório quando o estoque não é suficiente.
var ErrStockNotAvailable = errors.New("stock not available")

// StockRepo é o nosso Adapter para o estoque.
type StockRepo struct {
	rdb *redis.Client
	log *slog.Logger
}

// NewStockRepo é o construtor do nosso adapter.
func NewStockRepo(rdb *redis.Client, log *slog.Logger) *StockRepo {
	return &StockRepo{rdb: rdb, log: log}
}

const reserveStockScript = `
-- ARGV[1] = eventID
-- ARGV[2], ARGV[3]... = pares de ticketTypeID e quantity
local eventKey = "event:" .. ARGV[1] .. ":stock"

-- Passo 1: Verificar a disponibilidade de todos os itens
for i = 2, #ARGV, 2 do
    local ticketTypeID = ARGV[i]
    local requestedQuantity = tonumber(ARGV[i+1])
    local currentStock = tonumber(redis.call("HGET", eventKey, ticketTypeID))
    
    -- Se o campo não existir ou o estoque for insuficiente, retorna erro.
    if not currentStock or currentStock < requestedQuantity then
        return {err = "INSUFFICIENT_STOCK_FOR_" .. ticketTypeID}
    end
end

-- Passo 2: Se todos os itens estão disponíveis, decrementar todos
for i = 2, #ARGV, 2 do
    local ticketTypeID = ARGV[i]
    local requestedQuantity = tonumber(ARGV[i+1])
    redis.call("HINCRBY", eventKey, ticketTypeID, -requestedQuantity)
end

return "OK"
`

func (r *StockRepo) ReserveStock(ctx context.Context, eventID string, items []reservations.ReservationItem) error {

	args := make([]interface{}, 1+2*len(items))
	args[0] = eventID
	i := 1
	for _, item := range items {
		args[i] = item.TicketTypeID
		args[i+1] = item.Quantity
		i += 2
	}

	// Executa o script Lua.
	res, err := r.rdb.Eval(ctx, reserveStockScript, []string{}, args...).Result()
	if err != nil {
		// Se o Redis em si falhar.
		r.log.Error("failed to execute redis reserve stock script", "error", err)
		return fmt.Errorf("redis command failed: %w", err)
	}

	// Verifica o resultado do script.
	if resStr, ok := res.(string); !ok || resStr != "OK" {
		// Se o script retornou um erro (ex: estoque insuficiente),
		// o resultado será um mapa/tabela Lua, não a string "OK".
		r.log.Warn("stock reservation failed by script logic", "eventID", eventID, "result", res)
		return ErrStockNotAvailable
	}

	return nil
}

func (r *StockRepo) ReleaseStock(ctx context.Context, eventID string, items []reservations.ReservationItem) error {
	key := fmt.Sprintf("event:%s:stock", eventID)

	// Usamos um Pipeline para enviar todos os comandos de uma vez.
	// É mais eficiente e, embora não seja transacional como o MULTI,
	// é suficiente para esta operação de compensação.
	pipe := r.rdb.Pipeline()
	for _, item := range items {
		pipe.HIncrBy(ctx, key, item.TicketTypeID, int64(item.Quantity))
	}

	// Executa todos os comandos no pipeline.
	_, err := pipe.Exec(ctx)
	if err != nil {
		r.log.Error("failed to execute redis release stock pipeline", "eventID", eventID, "error", err)
		return fmt.Errorf("redis pipeline failed: %w", err)
	}

	return nil
}
