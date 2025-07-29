// File: internal/platform/events/envelope.go
package events

import (
	"time"
)

// EventEnvelope é a estrutura padrão para todos os eventos de domínio no sistema.
type EventEnvelope struct {
	EventID      string    `json:"eventId"`      // ID único desta instância do evento (para idempotência e tracing).
	EventName    string    `json:"eventName"`    // Nome do evento, ex: "ReservationCreated". Usado para roteamento.
	EventVersion string    `json:"eventVersion"` // Versão do schema do payload, ex: "1.0". Permite evolução.
	OccurredAt   time.Time `json:"occurredAt"`   // Timestamp de quando o evento ocorreu.
	Payload      any       `json:"payload"`      // Os dados específicos do evento. `any` para flexibilidade.
}
