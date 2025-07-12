package user

type User struct {
	ID            string
	Email         string
	PasswordHash  *string
	OAuthProvider *string
	OAuthID       *string
}
