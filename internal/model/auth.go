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
	User         *LoginUserResponse `json:"user"`
	Token        string             `json:"token"`
	RefreshToken string             `json:"refresh_token"`
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

type NewUserInfo struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
}

type UserAuthProvider struct {
	ID             int       // SERIAL PRIMARY KEY
	UserID         uuid.UUID // UUID as string
	AuthProvider   string    // e.g., "google"
	AuthProviderID string    // e.g., "google_user_123"

}
