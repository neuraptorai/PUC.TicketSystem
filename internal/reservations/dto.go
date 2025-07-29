// File: internal/reservations/dto.go
package reservations

import "time"

// ItemDTO para a requisição.
type ItemDTO struct {
	TicketTypeID string `json:"ticketTypeId"`
	Quantity     int    `json:"quantity"`
}

// CreateReservationRequestDTO define o corpo da requisição POST.
type CreateReservationRequestDTO struct {
	EventID string    `json:"eventId"`
	Items   []ItemDTO `json:"items"`
}

// ToDomainItems converte o DTO para o modelo de domínio.
func (dto *CreateReservationRequestDTO) ToDomainItems() []ReservationItem {
	items := make([]ReservationItem, len(dto.Items))
	for i, item := range dto.Items {
		items[i] = ReservationItem{
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
		}
	}
	return items
}

// ReservationResponseDTO define a resposta da API.
type ReservationResponseDTO struct {
	ReservationID string    `json:"reservationId"`
	Status        string    `json:"status"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// ToResponseDTO converte a entidade de domínio para o DTO de resposta.
func ToResponseDTO(r *Reservation) *ReservationResponseDTO {
	return &ReservationResponseDTO{
		ReservationID: r.ID,
		Status:        string(r.Status),
		ExpiresAt:     r.ExpiresAt,
	}
}
