package model

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

type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}
