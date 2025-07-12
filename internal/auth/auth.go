// File: internal/auth/auth.go
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/neuraptorai/PUC.TicketSystem/internal/user"
	"golang.org/x/oauth2"
)

// UserValidator define o contrato para validar as credenciais de um usuário.
type UserValidator interface {
	ValidateCredentials(email, password string) (userID string, err error)
	FindOrCreateByOAuth(provider, providerID, email string) (*user.User, error)
}

// Handler agrupa as dependências para os handlers de autenticação.
type Handler struct {
	log               *slog.Logger
	validator         UserValidator
	jwtSecretKey      string // A chave secreta agora é uma dependência.
	googleOAuthConfig *oauth2.Config
}

// NewHandler cria e retorna um novo Handler de autenticação.
func NewHandler(log *slog.Logger, v UserValidator, jwtSecretKey string, googleOAuthConfig *oauth2.Config) *Handler {
	return &Handler{
		log:               log,
		validator:         v,
		jwtSecretKey:      jwtSecretKey,
		googleOAuthConfig: googleOAuthConfig,
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

func (h *Handler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Gera um 'state' aleatório para proteção contra ataques CSRF.
	state, err := generateOauthState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}
	// Armazena o state em um cookie seguro.
	http.SetCookie(w, &http.Cookie{Name: "oauthstate", Value: state, Path: "/", HttpOnly: true, MaxAge: 3600})

	// Gera a URL de autorização do Google e redireciona o usuário.
	url := h.googleOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback é o endpoint que o Google chama após o usuário autorizar.
func (h *Handler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Valida o 'state' para prevenir CSRF.
	oauthState, _ := r.Cookie("oauthstate")
	if r.FormValue("state") != oauthState.Value {
		http.Error(w, "Invalid oauth state", http.StatusUnauthorized)
		return
	}

	// 2. Troca o código de autorização por um token.
	code := r.FormValue("code")
	token, err := h.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// 3. Usa o token para obter as informações do usuário do Google.
	client := h.googleOAuthConfig.Client(ctx, token)
	userInfoResp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer userInfoResp.Body.Close()

	var userInfo struct {
		ID    string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	// 4. Encontra ou cria o usuário em nosso banco de dados.
	appUser, err := h.validator.FindOrCreateByOAuth("google", userInfo.ID, userInfo.Email)
	if err != nil {
		http.Error(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	// 5. Gera nosso próprio JWT para o usuário.
	appToken, err := GenerateToken(appUser.ID, h.jwtSecretKey)
	if err != nil {
		http.Error(w, "Failed to generate app token", http.StatusInternalServerError)
		return
	}

	// 6. Retorna nosso JWT para o cliente (ou redireciona para o frontend).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TokenResponse{AccessToken: appToken})
}

// generateOauthState cria uma string aleatória para o parâmetro state do OAuth.
func generateOauthState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
