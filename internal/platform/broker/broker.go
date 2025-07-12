// File: internal/platform/broker/broker.go
package broker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Event define a estrutura padrão para todos os eventos publicados no sistema.
// Isso representa nossa "Linguagem Publicada", conforme ADR 003.
type Event struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"` // 'any' permite que qualquer struct de dados seja enviada.
}

// NewEvent cria uma nova instância de Event.
func NewEvent(source, eventType string, data any) *Event {
	return &Event{
		ID:        uuid.NewString(),
		Source:    source,
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// MessageBroker define a interface para publicar eventos.
// Qualquer sistema de mensageria (RabbitMQ, NATS, Kafka) deverá implementar isso.
type MessageBroker interface {
	Publish(ctx context.Context, topic string, event *Event) error
}

// LogBroker é uma implementação do MessageBroker que apenas loga os eventos.
// É perfeito para desenvolvimento e testes locais sem a necessidade de um broker real.
type LogBroker struct {
	log *slog.Logger
}

// NewLogBroker cria uma nova instância do LogBroker.
func NewLogBroker(log *slog.Logger) *LogBroker {
	return &LogBroker{log: log}
}

// Publish loga o evento que seria enviado para o tópico.
func (b *LogBroker) Publish(ctx context.Context, topic string, event *Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to marshal event for logging", "error", err)
		return err
	}

	b.log.InfoContext(
		ctx,
		"Event Published (mock)",
		"topic", topic,
		"event", string(eventJSON),
	)
	return nil
}
