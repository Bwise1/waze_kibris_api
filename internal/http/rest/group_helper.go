package rest

import (
	"context"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/jackc/pgx/v5/pgconn"
)

func (api *API) CreateGroupHelper(ctx context.Context, newGroup model.CommunityGroup) (model.CommunityGroup, string, string, error) {
	maxAttempts := 3
	for range maxAttempts {
		// Generate a new short code
		code := util.GenerateShortCode(6)
		newGroup.ShortCode = code

		group, err := api.CreateCommunityGroup(ctx, newGroup)
		if err == nil {
			return group, values.Created, "Group created successfully", nil
		}
		// Check for unique violation (Postgres error code "23505")
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" && pgErr.ConstraintName == "community_groups_short_code_key" {
			continue // Try again with a new code
		}
		return model.CommunityGroup{}, values.Error, "Failed to create group", err
	}
	return model.CommunityGroup{}, values.Error, "Could not generate unique group code", nil
}

func (api *API) SearchCommunityGroupsHelper(ctx context.Context) ([]model.CommunityGroup, string, string, error) {

	groups, err := api.SearchCommunityGroup(ctx)
	if err != nil {
		return []model.CommunityGroup{}, values.Error, "Failed to get groups", err
	}

	return groups, values.Success, "Groups returned successfully", nil
}
