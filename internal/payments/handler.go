// File: internal/payments/handler.go
package payments

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/broker"
	"github.com/neuraptorai/PUC.TicketSystem/internal/sales"
)

// EventHandler é o nosso consumidor de eventos para o contexto de pagamentos.
type EventHandler struct {
	log     *slog.Logger
	gateway PaymentGateway // Dependência da interface do gateway
}

// NewEventHandler cria um novo handler de eventos de pagamento.
func NewEventHandler(log *slog.Logger, gateway PaymentGateway) *EventHandler {
	return &EventHandler{
		log:     log,
		gateway: gateway,
	}
}

// HandleReservationCreated é o handler específico para o evento ReservaCriada.
// Sua responsabilidade é iniciar o fluxo de pagamento para a nova reserva. [cite: 16]
func (h *EventHandler) HandleReservationCreated(ctx context.Context, event *broker.Event) error {
	h.log.InfoContext(ctx, "Handling ReservationCreated event", "event_id", event.ID)

	var reservationData sales.ReservationData
	// Precisamos converter os dados do evento (que são do tipo 'any') para a struct correta.
	if dataBytes, err := json.Marshal(event.Data); err == nil {
		if err := json.Unmarshal(dataBytes, &reservationData); err != nil {
			return err
		}
	} else {
		return err
	}

	// 1. Chama a interface PaymentGateway para criar a intenção de pagamento. [cite: 18]
	paymentData := PaymentData{
		ReservationID: reservationData.ReservationID,
		Amount:        float64(reservationData.Quantity) * 50.0, // Preço fixo para simulação
		UserID:        reservationData.UserID,
	}

	result, err := h.gateway.CreatePaymentIntention(ctx, paymentData)
	if err != nil {
		h.log.ErrorContext(ctx, "Failed to create payment intention", "error", err, "reservation_id", reservationData.ReservationID)
		// Em um sistema real, aqui poderia haver uma lógica de retry ou mover para uma DLQ.
		return err
	}

	// 2. Persiste os dados de pagamento (ex: QR Code) para consulta. [cite: 19]
	// POR FAZER: Salvar result.QRCode e result.PaymentID no banco de dados,
	// associados à reserva. Por enquanto, vamos apenas logar.
	h.log.InfoContext(ctx, "Payment data persisted (simulated)", "payment_id", result.PaymentID, "reservation_id", reservationData.ReservationID)

	return nil
}
