// File: internal/auth/auth.go
package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// UserValidator define o contrato para validar as credenciais de um usuário.
type UserValidator interface {
	ValidateCredentials(email, password string) (userID string, err error)
}

// Handler agrupa as dependências para os handlers de autenticação.
type Handler struct {
	log          *slog.Logger
	validator    UserValidator
	jwtSecretKey string // A chave secreta agora é uma dependência.
}

// NewHandler cria e retorna um novo Handler de autenticação.
func NewHandler(log *slog.Logger, v UserValidator, jwtSecretKey string) *Handler {
	return &Handler{
		log:          log,
		validator:    v,
		jwtSecretKey: jwtSecretKey,
	}
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.validator.ValidateCredentials(creds.Email, creds.Password)
	if err != nil {
		h.log.Warn("Failed login attempt", "email", creds.Email, "error", err.Error())
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Passamos a chave secreta armazenada no handler para o gerador de token.
	tokenString, err := GenerateToken(userID, h.jwtSecretKey)
	if err != nil {
		h.log.Error("Failed to generate token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TokenResponse{AccessToken: tokenString})
}

// MockUserValidator (sem alterações)
type MockUserValidator struct{}

func (m *MockUserValidator) ValidateCredentials(email, password string) (string, error) {
	if email == "user@example.com" && password == "password123" {
		return "user-id-123", nil
	}
	return "", errors.New("user not found or password mismatch")
}
