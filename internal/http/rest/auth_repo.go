package rest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/jackc/pgx/v5"
)

// StoreVerificationToken(ctx context.Context, userID uuid.UUID, token string) error
// StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string) error
// RevokeRefreshToken(ctx context.Context, token string) error
// GetUserByRefreshToken(ctx context.Context, token string) (*User, error)

func (api *API) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	stmt := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	// err := api.Deps.DB.QueryRow(ctx, stmt, email).Scan(&exists)
	err := api.DB.QueryRow(ctx, stmt, email).Scan(&exists)
	if err != nil {
		log.Println("error checking email", err)
		return false, err
	}
	return exists, nil
}

func (api *API) CreateNewUserRepo(ctx context.Context, req model.User) error {
	stmt := `
        INSERT INTO users (
            id,
            email,
            auth_provider
        ) VALUES ($1, $2, $3)
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, stmt, req.ID, req.Email, req.AuthProvider)
	if err != nil {
		log.Println("error creating new user", err)
		return err
	}
	return nil
}

func (api *API) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	stmt := `-- name: get-user-by-email
		SELECT id, email FROM users WHERE email = $1`

	err := api.DB.QueryRow(ctx, stmt, email).Scan(
		&user.ID,
		&user.Email,
	)
	if err != nil {
		log.Println("error getting user by email", err)
		return model.User{}, err
	}
	return user, nil
}

func (api *API) GetUserByID(ctx context.Context, userID string) (model.User, error) {
	var user model.User
	stmt := `SELECT id, email, firstname, lastname, auth_provider, is_verified, preferred_language, created_at, updated_at FROM users WHERE id = $1`

	err := api.Deps.DB.Pool().QueryRow(ctx, stmt, userID).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.AuthProvider,
		&user.IsVerified,
		&user.PreferredLanguage,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		log.Println("error getting user by ID", err)
		return model.User{}, err
	}
	return user, nil
}

func (api *API) StoreVerificationCode(ctx context.Context, userID string, email string, code string, tokenType string, expiresAt time.Time) error {
	stmt := `
        INSERT INTO email_verifications (user_id, email, verification_code, type, expires_at)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := api.DB.Exec(ctx, stmt, userID, email, code, tokenType, expiresAt)
	if err != nil {
		log.Println("error storing verification code", err)
	}
	return err
}

// StoreRefreshToken stores the refresh token in the database
func (api *API) StoreRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	query := `
        INSERT INTO auth_tokens (user_id, token_type, token_value, expires_at, created_at)
        VALUES ($1, 'refresh', $2, $3, NOW())
    `
	_, err := api.DB.Exec(ctx, query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}
	return nil
}

func (api *API) ValidateRefreshToken(ctx context.Context, token string) error {
	query := `
        SELECT 1 FROM auth_tokens
        WHERE token_value = $1 AND token_type = 'refresh' AND is_revoked = FALSE AND expires_at > NOW()
    `
	var exists int
	err := api.DB.QueryRow(ctx, query, token).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("refresh token is invalid or expired")
		}
		return err
	}
	return nil
}

func (api *API) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `
        UPDATE auth_tokens
        SET is_revoked = TRUE
        WHERE token_value = $1
    `
	_, err := api.DB.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (api *API) VerifyCodeRepo(ctx context.Context, code string, tokenType string, email string) (string, error) {
	var userID string
	stmt := `SELECT user_id FROM email_verifications WHERE verification_code = $1 AND type = $2 AND email= $3 AND expires_at > NOW()`

	err := api.Deps.DB.Pool().QueryRow(ctx, stmt, code, tokenType, email).Scan(&userID)
	if err != nil {
		log.Println("error verifying code", err)
		return "", err
	}
	return userID, nil
}

func (api *API) UpdateEmailVerifiedStatus(ctx context.Context, userID string) error {
	stmt := `UPDATE users SET is_verified = TRUE WHERE id = $1`

	_, err := api.Deps.DB.Pool().Exec(ctx, stmt, userID)
	if err != nil {
		log.Println("error updating email verification status", err)
		return err
	}
	return nil
}

func (api *API) verifyTokenRepo() (*string, error) {
	return nil, nil
}
