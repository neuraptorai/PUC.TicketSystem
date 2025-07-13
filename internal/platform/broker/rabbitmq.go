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
	// --- Configuração da Dead-Letter Queue (DLQ) ---
	dlqExchange := exchangeName + ".dlx"
	dlqQueue := queueName + ".dlq"

	// Declara o exchange da DLQ
	err = ch.ExchangeDeclare(dlqExchange, "direct", true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to declare DLQ exchange", "error", err, "exchange", dlqExchange)
		return
	}

	// Declara a fila da DLQ
	_, err = ch.QueueDeclare(dlqQueue, true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to declare DLQ", "error", err, "queue", dlqQueue)
		return
	}

	// Faz o bind da fila DLQ com o exchange DLQ
	err = ch.QueueBind(dlqQueue, routingKey, dlqExchange, false, nil)
	if err != nil {
		b.log.Error("Failed to bind DLQ", "error", err, "queue", dlqQueue)
		return
	}
	// --- Configuração da Fila Principal ---
	err = ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to declare main exchange", "error", err, "exchange", exchangeName)
		return
	}

	// Declara a fila principal com os argumentos para apontar para a DLQ
	args := amqp091.Table{
		"x-dead-letter-exchange":    dlqExchange,
		"x-dead-letter-routing-key": routingKey,
	}
	q, err := ch.QueueDeclare(queueName, true, false, false, false, args)
	if err != nil {
		b.log.Error("Failed to declare main queue", "error", err, "queue", queueName)
		return
	}

	err = ch.QueueBind(q.Name, routingKey, exchangeName, false, nil)
	if err != nil {
		b.log.Error("Failed to bind main queue", "error", err, "queue", queueName)
		return
	}

	// Altera o consumo para 'autoAck: false' para Acknowledgment manual
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		b.log.Error("Failed to register a consumer", "error", err)
		return
	}

	b.log.Info("Consumer started. Waiting for messages.", "queue", q.Name, "routing_key", routingKey)

	for msg := range msgs {
		var event Event
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			b.log.Error("Failed to unmarshal event (poison pill). Sending to DLQ.", "error", err)
			// Rejeita a mensagem e NÃO a recoloca na fila, fazendo com que vá para a DLQ.
			_ = msg.Nack(false, false)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = handler(ctx, &event)
		cancel()

		if err != nil {
			b.log.Error("Event handler failed. Sending to DLQ.", "error", err, "event_id", event.ID)
			// Rejeita a mensagem e NÃO a recoloca na fila.
			_ = msg.Nack(false, false)
		} else {
			// Confirma que a mensagem foi processada com sucesso.
			_ = msg.Ack(false)
		}
	}
}

func (b *RabbitMQBroker) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
}
