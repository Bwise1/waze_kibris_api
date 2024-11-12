package model

import "github.com/google/uuid"

type User struct {
	ID                uuid.UUID `json:"id"`
	FirstName         string    `json:"firstname"`
	LastName          string    `json:"lastname"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"password_hash"`
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsDeleted         bool      `json:"is_deleted"`
	AuthProvider      string    `json:"auth_provider"`
	PreferredLanguage string    `json:"preferred_language"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}
