// File: internal/platform/broker/broker.go
package broker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ... (struct Event e func NewEvent sem alterações) ...
type Event struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"` // 'any' permite que qualquer struct de dados seja enviada.
}

func NewEvent(source, eventType string, data any) *Event {
	return &Event{
		ID:        uuid.NewString(),
		Source:    source,
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// EventHandlerFunc é um tipo para as funções que manipulam eventos.
type EventHandlerFunc func(context.Context, *Event) error

type MessageBroker interface {
	Publish(ctx context.Context, topic string, event *Event) error
	// Subscribe permite que um handler se "inscreva" em um tópico.
	Subscribe(topic string, handler EventHandlerFunc)
}

type LogBroker struct {
	log      *slog.Logger
	handlers map[string][]EventHandlerFunc // Mapeia tópicos para uma lista de handlers
	mu       sync.Mutex                    // Protege o acesso concorrente ao mapa de handlers
}

func NewLogBroker(log *slog.Logger) *LogBroker {
	return &LogBroker{
		log:      log,
		handlers: make(map[string][]EventHandlerFunc),
	}
}

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

	// Simula o consumo do evento despachando para os handlers inscritos.
	b.dispatch(ctx, topic, event)

	return nil
}

// Subscribe adiciona um handler a um tópico.
func (b *LogBroker) Subscribe(topic string, handler EventHandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}

// dispatch encontra os handlers para um tópico e os executa.
func (b *LogBroker) dispatch(ctx context.Context, topic string, event *Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if handlers, found := b.handlers[topic]; found {
		for _, handler := range handlers {
			// Executamos cada handler em sua própria goroutine para não bloquear a publicação.
			go func(h EventHandlerFunc) {
				if err := h(context.Background(), event); err != nil {
					b.log.ErrorContext(ctx, "Event handler failed", "error", err, "topic", topic, "event_id", event.ID)
				}
			}(handler)
		}
	}
}
