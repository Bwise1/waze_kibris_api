package rest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (api *API) CreateCommunityGroup(ctx context.Context, group model.CommunityGroup) (model.CommunityGroup, error) {
	var createdGroup model.CommunityGroup

	err := api.Deps.DB.RunInTx(ctx, func(tx pgx.Tx) error {
		// Build dynamic insert as before
		columns := []string{"id", "name", "description", "destination_place_id", "destination_name",
			"creator_id", "is_deleted", "created_at", "updated_at"}
		values := []string{"$1", "$2", "$3", "$4", "$5", "$6", "$7", "$8", "$9"}

		group.ID = uuid.New()
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()

		args := []interface{}{
			group.ID, group.Name, group.Description, group.DestinationPlaceID,
			group.DestinationName, group.CreatorID, group.IsDeleted, group.CreatedAt, group.UpdatedAt,
		}

		paramCount := 9

		if group.GroupType != "" {
			paramCount++
			columns = append(columns, "group_type")
			values = append(values, fmt.Sprintf("$%d", paramCount))
			args = append(args, group.GroupType)
		}
		if group.Visibility != "" {
			paramCount++
			columns = append(columns, "visibility")
			values = append(values, fmt.Sprintf("$%d", paramCount))
			args = append(args, group.Visibility)
		}
		if group.DestinationLocation != nil && *group.DestinationLocation != "" {
			paramCount++
			columns = append(columns, "destination_location")
			values = append(values, fmt.Sprintf("ST_GeomFromText($%d, 4326)", paramCount))
			args = append(args, *group.DestinationLocation)
		}
		if group.IconURL != nil {
			paramCount++
			columns = append(columns, "icon_url")
			values = append(values, fmt.Sprintf("$%d", paramCount))
			args = append(args, group.IconURL)
		}
		if group.ShortCode != "" {
			paramCount++
			columns = append(columns, "short_code")
			values = append(values, fmt.Sprintf("$%d", paramCount))
			args = append(args, group.ShortCode)
		}

		query := fmt.Sprintf(`
            INSERT INTO community_groups (%s)
            VALUES (%s)
            RETURNING id, name, description, group_type, destination_place_id, destination_name,
                      ST_AsText(destination_location), visibility, creator_id, icon_url, member_count,
                      last_message_at, is_deleted, created_at, updated_at, short_code
        `, strings.Join(columns, ", "), strings.Join(values, ", "))

		err := tx.QueryRow(ctx, query, args...).Scan(
			&createdGroup.ID, &createdGroup.Name, &createdGroup.Description, &createdGroup.GroupType, &createdGroup.DestinationPlaceID,
			&createdGroup.DestinationName, &createdGroup.DestinationLocation, &createdGroup.Visibility, &createdGroup.CreatorID,
			&createdGroup.IconURL, &createdGroup.MemberCount, &createdGroup.LastMessageAt, &createdGroup.IsDeleted,
			&createdGroup.CreatedAt, &createdGroup.UpdatedAt, &createdGroup.ShortCode,
		)
		if err != nil {
			return err
		}

		// Insert creator into group_memberships as admin
		_, err = tx.Exec(ctx, `
            INSERT INTO group_memberships (group_id, user_id, role, joined_at, updated_at)
            VALUES ($1, $2, 'admin', NOW(), NOW())
        `, createdGroup.ID, createdGroup.CreatorID)
		return err
	})

	if err != nil {
		log.Println("error creating new group chat or adding creator to membership", err)
		return model.CommunityGroup{}, err
	}

	return createdGroup, nil
}

func (api *API) GetCommunityGroupByID(ctx context.Context, groupID uuid.UUID) (model.CommunityGroup, error) {
	query := `
        SELECT id, name, description, group_type, destination_place_id, destination_name,
               ST_AsText(destination_location), visibility, creator_id, icon_url, member_count,
               last_message_at, is_deleted, created_at, updated_at,short_code
        FROM community_groups
        WHERE id = $1 AND is_deleted = FALSE
    `

	var group model.CommunityGroup
	err := api.Deps.DB.Pool().QueryRow(ctx, query, groupID).Scan(
		&group.ID, &group.Name, &group.Description, &group.GroupType, &group.DestinationPlaceID,
		&group.DestinationName, &group.DestinationLocation, &group.Visibility, &group.CreatorID,
		&group.IconURL, &group.MemberCount, &group.LastMessageAt, &group.IsDeleted,
		&group.CreatedAt, &group.UpdatedAt, &group.ShortCode,
	)

	return group, err
}

func (api *API) SoftDeleteCommunityGroup(ctx context.Context, groupID uuid.UUID) error {
	query := `
        UPDATE community_groups
        SET is_deleted = TRUE, deleted_at = NOW()
        WHERE id = $1
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, query, groupID)
	return err
}

func (api *API) UpdateCommunityGroup(ctx context.Context, group model.CommunityGroup) error {
	query := `
        UPDATE community_groups
        SET name = $1, description = $2, group_type = $3, destination_place_id = $4,
            destination_name = $5, destination_location = ST_GeomFromText($6, 4326),
            visibility = $7, icon_url = $8, updated_at = NOW()
        WHERE id = $9 AND is_deleted = FALSE
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, query,
		group.Name, group.Description, group.GroupType, group.DestinationPlaceID,
		group.DestinationName, group.DestinationLocation, group.Visibility,
		group.IconURL, group.ID,
	)
	return err
}

func (api *API) SearchCommunityGroup(ctx context.Context) ([]model.CommunityGroup, error) {

	query := `
		SELECT id, name, creator_id, short_code
		FROM community_groups
	`

	rows, err := api.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying nearby reports: %w", err)
	}
	defer rows.Close()

	var groups []model.CommunityGroup

	for rows.Next() {
		var group model.CommunityGroup
		err := rows.Scan(
			&group.ID, &group.Name, &group.CreatorID, &group.ShortCode,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning groups: %w", err)
		}

		// report.Distance = distance // Add distance to report model
		groups = append(groups, group)
	}

	return groups, nil
}

func (api *API) GetCommunityGroupByShortCode(ctx context.Context, shortCode string) (model.CommunityGroup, error) {
	query := `
        SELECT id, name, description, group_type, destination_place_id, destination_name,
               ST_AsText(destination_location), visibility, creator_id, icon_url, member_count,
               last_message_at, is_deleted, created_at, updated_at, short_code
        FROM community_groups
        WHERE short_code = $1 AND is_deleted = FALSE
    `
	var group model.CommunityGroup
	err := api.Deps.DB.Pool().QueryRow(ctx, query, shortCode).Scan(
		&group.ID, &group.Name, &group.Description, &group.GroupType, &group.DestinationPlaceID,
		&group.DestinationName, &group.DestinationLocation, &group.Visibility, &group.CreatorID,
		&group.IconURL, &group.MemberCount, &group.LastMessageAt, &group.IsDeleted,
		&group.CreatedAt, &group.UpdatedAt, &group.ShortCode,
	)
	return group, err
}
