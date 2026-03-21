package rest

import (
	"context"
	"fmt"
)

// UpsertFCMToken stores or updates a device token for the user (multi-device).
func (api *API) UpsertFCMToken(ctx context.Context, userID, token, platform string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}
	q := `
		INSERT INTO user_fcm_tokens (user_id, token, platform, updated_at)
		VALUES ($1::uuid, $2, $3, now())
		ON CONFLICT (user_id, token) DO UPDATE SET
			platform = EXCLUDED.platform,
			updated_at = now()
	`
	_, err := api.DB.Exec(ctx, q, userID, token, platform)
	return err
}

// DeleteFCMToken removes one device token (e.g. logout on that device).
func (api *API) DeleteFCMToken(ctx context.Context, userID, token string) error {
	q := `DELETE FROM user_fcm_tokens WHERE user_id = $1::uuid AND token = $2`
	_, err := api.DB.Exec(ctx, q, userID, token)
	return err
}

// DeleteAllFCMTokensForUser removes every token for the user (full logout).
func (api *API) DeleteAllFCMTokensForUser(ctx context.Context, userID string) error {
	_, err := api.DB.Exec(ctx, `DELETE FROM user_fcm_tokens WHERE user_id = $1::uuid`, userID)
	return err
}

// GetFCMTokensForUser returns all FCM registration tokens for sending notifications.
func (api *API) GetFCMTokensForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := api.DB.Query(ctx, `SELECT token FROM user_fcm_tokens WHERE user_id = $1::uuid`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
