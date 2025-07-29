// File: internal/adapters/rabbitmq/publisher.go
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/events"
	"github.com/rabbitmq/amqp091-go"
)

// EventPublisher é a implementação concreta para RabbitMQ.
type EventPublisher struct {
	channel        *amqp091.Channel
	log            *slog.Logger
	domainExchange string // Nome da exchange principal da nossa aplicação
}

// NewEventPublisher é o construtor do nosso adapter.
func NewEventPublisher(channel *amqp091.Channel, log *slog.Logger, domainExchange string) *EventPublisher {
	return &EventPublisher{
		channel:        channel,
		log:            log,
		domainExchange: domainExchange,
	}
}

// Publish implementa a interface `EventPublisher`.
func (p *EventPublisher) Publish(ctx context.Context, eventData any) error {
	// 1. Valida se o evento é do nosso tipo esperado.
	event, ok := eventData.(*events.EventEnvelope)
	if !ok {
		return fmt.Errorf("eventData is not of type *events.EventEnvelope")
	}

	// 2. Serializa o evento para JSON.
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// 3. Define a chave de roteamento. Ex: "reservations.created"
	// Isso permite que os consumidores filtrem as mensagens que querem receber.
	routingKey := fmt.Sprintf("%s.%s", "reservations", "created") // Exemplo; poderia ser mais dinâmico
	if event.EventName == "ReservationCreated" {
		routingKey = "reservations.created"
	}

	p.log.Info("publishing event", "exchange", p.domainExchange, "routingKey", routingKey, "eventId", event.EventID)

	// 4. Publica a mensagem na Exchange.
	err = p.channel.PublishWithContext(ctx,
		p.domainExchange, // exchange
		routingKey,       // routing key
		false,            // mandatory
		false,            // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent, // Garante que a mensagem sobreviva a reinícios do broker
			Body:         body,
			AppId:        "ticket-sales-system", // Identificador da aplicação que publicou
			Timestamp:    event.OccurredAt,
			MessageId:    event.EventID,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message to rabbitmq: %w", err)
	}

	return nil
}
