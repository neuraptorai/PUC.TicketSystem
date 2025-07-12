// File: internal/payments/adapter.go
package payments

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// PaymentData contém as informações necessárias para criar uma intenção de pagamento.
type PaymentData struct {
	ReservationID string
	Amount        float64 // Simulação, o valor viria da reserva/evento
	UserID        string
}

// PaymentIntentionResult contém o resultado da criação de uma intenção de pagamento.
// Conforme ADR 005, um dos resultados é o QR Code para o front-end consultar.
type PaymentIntentionResult struct {
	PaymentID string
	QRCode    string
	Status    string
}

// PaymentGateway é a nossa interface (Port) que define o contrato
// para interagir com um gateway de pagamento externo. [cite: 86]
// Ela dita o que nosso sistema precisa, não o que o sistema externo oferece. [cite: 88]
type PaymentGateway interface {
	CreatePaymentIntention(ctx context.Context, data PaymentData) (*PaymentIntentionResult, error)
}

// fakePaymentGateway é uma implementação "fake" da nossa interface.
// Ele simula o comportamento de um gateway como o Mercado Pago e permite
// testar nossa lógica de negócio de forma isolada. [cite: 100]
type fakePaymentGateway struct {
	log *slog.Logger
}

// NewFakePaymentGateway cria uma nova instância do nosso adapter fake.
func NewFakePaymentGateway(log *slog.Logger) PaymentGateway {
	return &fakePaymentGateway{log: log}
}

// CreatePaymentIntention simula a chamada para um serviço externo.
func (f *fakePaymentGateway) CreatePaymentIntention(ctx context.Context, data PaymentData) (*PaymentIntentionResult, error) {
	f.log.InfoContext(ctx, "Simulating call to external payment gateway", "reservation_id", data.ReservationID, "amount", data.Amount)

	// Simula uma pequena latência de rede
	time.Sleep(150 * time.Millisecond)

	// Simula a criação de um QR Code e um ID de pagamento
	paymentID := fmt.Sprintf("pay_%s", data.ReservationID)
	qrCodeData := fmt.Sprintf("QR_CODE_DATA_FOR_%s", paymentID)

	result := &PaymentIntentionResult{
		PaymentID: paymentID,
		QRCode:    qrCodeData,
		Status:    "PENDING",
	}

	f.log.InfoContext(ctx, "Payment intention created successfully (fake)", "payment_id", result.PaymentID)
	return result, nil
}
