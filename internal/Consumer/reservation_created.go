// File: internal/consumers/reservation_created.go
package consumers

import (
	"context"
	"log/slog"
)

type ReservationCreatedConsumer struct {
	log *slog.Logger
	// paymentsSvc *payments.Service // Depende do serviço de pagamentos!
	// ... e da conexão com a fila ...
}

func (c *ReservationCreatedConsumer) Start(ctx context.Context) {
	// Aqui entra a lógica de se conectar à fila "payments.reservation_created.queue"
	// e de entrar em um loop para consumir mensagens.
	// Ao receber uma mensagem, ele a decodifica e chama c.paymentsSvc.ProcessReservation(ctx, payload)
}
