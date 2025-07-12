// File: internal/auth/jwt.go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims define a estrutura do payload (claims) do nosso JWT.
// Inclui claims registrados (exp, iat) e claims customizados (userID).
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken cria um novo token JWT assinado para um determinado ID de usuário.
// O token de acesso terá uma vida útil curta, conforme ADR 006.
func GenerateToken(userID, secretKey string) (string, error) {
	// Define a duração do token de acesso (ex: 15 minutos).
	expirationTime := time.Now().Add(15 * time.Minute) //

	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			// Define o tempo de expiração do token.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			// Define o tempo de emissão do token.
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// Define o emissor do token.
			Issuer: "puc.ticketsystem",
		},
	}

	// Cria o token com as claims e o método de assinatura HS256.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assina o token com a nossa chave secreta e obtém a string completa.
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
