package rest

import (
	"context"

	"github.com/bwise1/waze_kibris/internal/model"
)

func (api *API) GetUserProfileByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	stmt := `SELECT id, email, firstname, lastname, auth_provider, is_verified, preferred_language, created_at, updated_at FROM users WHERE id = $1`

	err := api.Deps.DB.Pool().QueryRow(ctx, stmt, id).Scan(
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
		return model.User{}, err
	}
	return user, nil
}

func (api *API) UpdateUserRepo(ctx context.Context, user model.User) error {
	stmt := `
        UPDATE users
        SET firstname = $2, lastname = $3, updated_at = NOW()
        WHERE id = $1
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, stmt, user.ID, user.FirstName, user.LastName)
	if err != nil {
		return err
	}
	return nil
}

func (api *API) ChangePasswordRepo(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Implement password change logic here
	return nil
}

func (api *API) UpdateLanguageRepo(ctx context.Context, userID, language string) error {
	stmt := `
        UPDATE users
        SET preferred_language = $2, updated_at = NOW()
        WHERE id = $1
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, stmt, userID, language)
	if err != nil {
		return err
	}
	return nil
}

func (api *API) DeleteUserRepo(ctx context.Context, userID string) error {
	stmt := `DELETE FROM users WHERE id = $1`

	_, err := api.Deps.DB.Pool().Exec(ctx, stmt, userID)
	if err != nil {
		return err
	}
	return nil
}
