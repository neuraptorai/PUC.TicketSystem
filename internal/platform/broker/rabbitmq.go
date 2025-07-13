// File: internal/platform/broker/rabbitmq.go
package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	log  *slog.Logger
	conn *amqp091.Connection
}

func NewRabbitMQBroker(dsn string, log *slog.Logger) (*RabbitMQBroker, error) {
	conn, err := amqp091.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &RabbitMQBroker{
		log:  log,
		conn: conn,
	}, nil
}

// Publish publica um evento em um tópico (que será um exchange no RabbitMQ).
func (b *RabbitMQBroker) Publish(ctx context.Context, topic string, event *Event) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Garante que o exchange do tipo 'topic' exista.
	err = ch.ExchangeDeclare(
		topic,   // nome do exchange
		"topic", // tipo
		true,    // durável
		false,   // auto-delete
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(ctx,
		topic,      // exchange
		event.Type, // routing key (usaremos o tipo do evento como chave de roteamento)
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// StartConsumer inicia um worker para consumir mensagens de uma fila.
func (b *RabbitMQBroker) StartConsumer(exchangeName, queueName, routingKey string, handler EventHandlerFunc) {
	ch, err := b.conn.Channel()
	if err != nil {
		b.log.Error("Failed to open a channel for consumer", "error", err)
		return
	}
	// Não fechamos o canal aqui, pois ele precisa ficar aberto para o consumidor.

	err = ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to declare an exchange for consumer", "error", err)
		return
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to declare a queue", "error", err)
		return
	}

	err = ch.QueueBind(q.Name, routingKey, exchangeName, false, nil)
	if err != nil {
		b.log.Error("Failed to bind a queue", "error", err)
		return
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to register a consumer", "error", err)
		return
	}

	b.log.Info("Consumer started. Waiting for messages.", "queue", q.Name, "routing_key", routingKey)

	// Loop infinito para processar mensagens
	for msg := range msgs {
		var event Event
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			b.log.Error("Failed to unmarshal event from message", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := handler(ctx, &event); err != nil {
			b.log.Error("Event handler failed", "error", err, "event_id", event.ID)
		}
		cancel()
	}
}

func (b *RabbitMQBroker) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
}
