// File: internal/payments/handler.go
package payments

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/neuraptorai/PUC.TicketSystem/internal/platform/broker"
	"github.com/neuraptorai/PUC.TicketSystem/internal/sales"
)

// EventHandler agora também precisa de um broker para publicar eventos.
type EventHandler struct {
	log     *slog.Logger
	gateway PaymentGateway
	broker  broker.MessageBroker
}

// NewEventHandler agora recebe a dependência do broker.
func NewEventHandler(log *slog.Logger, gateway PaymentGateway, broker broker.MessageBroker) *EventHandler {
	return &EventHandler{
		log:     log,
		gateway: gateway,
		broker:  broker,
	}
}

// WebhookPayload simula a notificação recebida de um gateway externo.
type WebhookPayload struct {
	PaymentID     string  `json:"payment_id"`
	ReservationID string  `json:"reservation_id"`
	Status        string  `json:"status"` // ex: "APPROVED", "REJECTED"
	Amount        float64 `json:"amount"`
}

// --- Handlers de API (HTTP) ---

// HandleWebhook é o handler para POST /api/v1/payments/webhook.
// Ele apenas recebe a notificação, a valida minimamente e publica um evento interno.
func (h *EventHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	h.log.InfoContext(ctx, "Webhook received", "payment_id", payload.PaymentID, "status", payload.Status)

	// Publica um evento interno para desacoplar a lógica de negócio do protocolo HTTP.
	paymentEvent := broker.NewEvent("payment-webhook", "PaymentProcessed", payload)
	topic := "payments.processed"
	if err := h.broker.Publish(ctx, topic, paymentEvent); err != nil {
		h.log.ErrorContext(ctx, "Failed to publish payment processed event", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Responde imediatamente ao webhook com 202 Accepted. [cite: 19]
	w.WriteHeader(http.StatusAccepted)
}

// --- Handlers de Eventos (Workers) ---

// HandleReservationCreated (sem alterações)
func (h *EventHandler) HandleReservationCreated(ctx context.Context, event *broker.Event) error {
	h.log.InfoContext(ctx, "Handling ReservationCreated event", "event_id", event.ID)

	var reservationData sales.ReservationData
	if dataBytes, err := json.Marshal(event.Data); err == nil {
		if err := json.Unmarshal(dataBytes, &reservationData); err != nil {
			return err
		}
	} else {
		return err
	}

	paymentData := PaymentData{
		ReservationID: reservationData.ReservationID,
		Amount:        float64(reservationData.Quantity) * 50.0,
		UserID:        reservationData.UserID,
	}

	result, err := h.gateway.CreatePaymentIntention(ctx, paymentData)
	if err != nil {
		h.log.ErrorContext(ctx, "Failed to create payment intention", "error", err, "reservation_id", reservationData.ReservationID)
		return err
	}

	h.log.InfoContext(ctx, "Payment data persisted (simulated)", "payment_id", result.PaymentID, "reservation_id", reservationData.ReservationID)
	return nil
}

// HandlePaymentProcessed é o handler para o evento interno PaymentProcessed.
// Aqui reside a lógica de negócio crítica para finalizar a compra.
func (h *EventHandler) HandlePaymentProcessed(ctx context.Context, event *broker.Event) error {
	h.log.InfoContext(ctx, "Handling PaymentProcessed event", "event_id", event.ID)

	var payload WebhookPayload
	if dataBytes, err := json.Marshal(event.Data); err == nil {
		if err := json.Unmarshal(dataBytes, &payload); err != nil {
			return err
		}
	} else {
		return err
	}

	// Se o pagamento foi aprovado, efetivamos a compra.
	if payload.Status == "APPROVED" {
		h.log.InfoContext(ctx, "Payment APPROVED. Finalizing order.", "reservation_id", payload.ReservationID)
		// POR FAZER: Iniciar transação de banco de dados aqui. [cite: 62]
		// 1. Atualizar o status da Reserva para CONVERTIDA. [cite: 63]
		h.log.InfoContext(ctx, "Updating reservation status to CONVERTED (simulated)", "reservation_id", payload.ReservationID)
		// 2. Criar registros nas tabelas pedidos e itens_pedido. [cite: 63]
		h.log.InfoContext(ctx, "Creating order and order_items records (simulated)", "reservation_id", payload.ReservationID)
		// 3. Se a transação for bem-sucedida (COMMIT), publicar o evento final PedidoConfirmado. [cite: 64]
		h.log.InfoContext(ctx, "Publishing PedidoConfirmado event (simulated)")

	} else {
		// Se o pagamento foi recusado, devolvemos os ingressos ao estoque.
		h.log.WarnContext(ctx, "Payment REJECTED. Returning tickets to stock.", "reservation_id", payload.ReservationID)
		// POR FAZER: Atualizar status da reserva e reverter a operação no Redis. [cite: 65]
	}

	// Idempotência: Este handler deve ser idempotente.
	// Em um sistema real, verificaríamos se o pedido para esta reserva já foi processado
	// antes de executar a lógica novamente.

	return nil
}
