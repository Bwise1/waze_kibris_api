package model

import (
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResendCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyCodeRequest struct {
	Code  string `json:"code" validate:"required"`
	Type  string `json:"type" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type VerifyCodeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type LoginUserResponse struct {
	ID                uuid.UUID `json:"id"`
	FirstName         *string   `json:"firstname,omitempty"`
	LastName          *string   `json:"lastname,omitempty"`
	Username          *string   `json:"username,omitempty"`
	Email             string    `json:"email"`
	IsVerified        bool      `json:"is_verified"`
	PreferredLanguage *string   `json:"preferred_language,omitempty"`
}

type LoginResponse struct {
	User  *LoginUserResponse `json:"user"`
	Token string             `json:"token"`
}
