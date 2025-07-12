// File: internal/user/repository.go
package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

// Repository encapsula o acesso ao banco de dados para usuários.
type Repository struct {
	log *slog.Logger
	db  *pgxpool.Pool
}

// NewRepository cria uma nova instância do repositório de usuário.
func NewRepository(log *slog.Logger, db *pgxpool.Pool) *Repository {
	return &Repository{log: log, db: db}
}

// ValidateCredentials verifica o e-mail e a senha de um usuário no banco.
// No futuro, a senha deve ser verificada usando bcrypt.
func (r *Repository) ValidateCredentials(email, password string) (userID string, err error) {
	ctx := context.Background()
	query := `SELECT id, password_hash FROM users WHERE email = $1`

	var passwordHash string
	err = r.db.QueryRow(ctx, query, email).Scan(&userID, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	// POR FAZER: Comparar a senha fornecida com o hash usando bcrypt.
	// Por simplicidade, vamos comparar a senha em texto plano por enquanto.
	if password != passwordHash {
		return "", ErrUserNotFound
	}

	return userID, nil
}

func (r *Repository) FindOrCreateByOAuth(provider, providerID, email string) (*User, error) {
	ctx := context.Background()
	query := `SELECT id, email FROM users WHERE oauth_provider = $1 AND oauth_id = $2`

	var user User
	err := r.db.QueryRow(ctx, query, provider, providerID).Scan(&user.ID, &user.Email)

	if err == nil {
		// Usuário encontrado, retorna.
		r.log.InfoContext(ctx, "OAuth user found", "email", email, "provider", provider)
		return &user, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		// Erro inesperado no banco de dados.
		return nil, err
	}

	// Usuário não encontrado (pgx.ErrNoRows), vamos criá-lo.
	r.log.InfoContext(ctx, "OAuth user not found, creating new user", "email", email, "provider", provider)
	createQuery := `
		INSERT INTO users (email, oauth_provider, oauth_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var newUserID string
	err = r.db.QueryRow(ctx, createQuery, email, provider, providerID).Scan(&newUserID)
	if err != nil {
		// Pode haver um erro de violação de constraint (ex: email duplicado)
		r.log.ErrorContext(ctx, "Failed to create OAuth user", "error", err)
		return nil, err
	}

	return &User{ID: newUserID, Email: email}, nil
}
