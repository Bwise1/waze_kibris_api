package model

import (
	"time"

	"github.com/google/uuid"
)

// type User struct {
// 	ID                uuid.UUID `json:"id"`
// 	FirstName         string    `json:"firstname"`
// 	LastName          string    `json:"lastname"`
// 	Username          string    `json:"username"`
// 	Email             string    `json:"email"`
// 	PasswordHash      string    `json:"password_hash"`
// 	IsDeleted         bool      `json:"is_deleted"`
// 	AuthProvider      string    `json:"auth_provider"`
// 	PreferredLanguage string    `json:"preferred_language"`
// 	CreatedAt         string    `json:"created_at"`
// 	UpdatedAt         string    `json:"updated_at"`
// }

type User struct {
	ID                uuid.UUID `json:"id"`
	FirstName         *string   `json:"firstname,omitempty"`
	LastName          *string   `json:"lastname,omitempty"`
	Username          *string   `json:"username,omitempty"`
	Email             string    `json:"email"`
	IsDeleted         bool      `json:"is_deleted,omitempty"`
	AuthProvider      string    `json:"auth_provider,omitempty"`
	IsVerified        bool      `json:"is_verified"`
	PreferredLanguage *string   `json:"preferred_language,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type UpdateLanguageRequest struct {
	Language string `json:"language" validate:"required"`
}
