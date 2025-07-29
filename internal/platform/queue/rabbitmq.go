// File: internal/platform/queue/rabbitmq.go (NOVO PACOTE)
package queue

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go" // Driver oficial do RabbitMQ
)

// RabbitMQConnection contém o estado da nossa conexão.
type RabbitMQConnection struct {
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

// NewRabbitMQConnection estabelece a conexão e um canal.
func NewRabbitMQConnection(dsn string) (*RabbitMQConnection, error) {
	conn, err := amqp091.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &RabbitMQConnection{Conn: conn, Channel: ch}, nil
}

// SetupDomainExchanges declara as exchanges que nossa aplicação usará.
// Esta função deve ser chamada na inicialização para garantir que a infraestrutura exista.
func (r *RabbitMQConnection) SetupDomainExchanges(exchangeName string) error {
	err := r.Channel.ExchangeDeclare(
		exchangeName, // nome da exchange
		"topic",      // tipo: topic exchange é a mais flexível
		true,         // durable: a exchange sobrevive a reinícios do broker
		false,        // auto-deleted: não deletar quando não houver mais filas ligadas
		false,        // internal: não, ela aceita publicações de fora
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange '%s': %w", exchangeName, err)
	}
	return nil
}

func (r *RabbitMQConnection) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
