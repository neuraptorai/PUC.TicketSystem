// File: internal/auth/jwt.go
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ... (struct JWTClaims e func GenerateToken permanecem os mesmos) ...
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, secretKey string) (string, error) {
	expirationTime := time.Now().Add(60 * time.Minute) // Aumentei para 1h para facilitar os testes

	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "puc.ticketsystem",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// contextKey é um tipo customizado para evitar colisões de chaves no contexto.
type contextKey string

// UserIDKey é a chave usada para armazenar o UserID no contexto da requisição.
const UserIDKey contextKey = "userID"

// Middleware valida o token JWT do cabeçalho Authorization.
func Middleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(secretKey), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Adiciona o userID ao contexto da requisição para uso posterior.
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
