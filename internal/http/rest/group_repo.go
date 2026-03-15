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
        SELECT id, name, description, group_type, destination_place_id, destination_name,
               ST_AsText(destination_location), visibility, creator_id, icon_url, member_count,
               last_message_at, is_deleted, created_at, updated_at, short_code
        FROM community_groups
        WHERE is_deleted = FALSE
    `
	rows, err := api.Deps.DB.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying community groups: %w", err)
	}
	defer rows.Close()

	var groups []model.CommunityGroup
	for rows.Next() {
		var group model.CommunityGroup
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.GroupType, &group.DestinationPlaceID,
			&group.DestinationName, &group.DestinationLocation, &group.Visibility, &group.CreatorID,
			&group.IconURL, &group.MemberCount, &group.LastMessageAt, &group.IsDeleted,
			&group.CreatedAt, &group.UpdatedAt, &group.ShortCode,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning groups: %w", err)
		}
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

func (api *API) GetGroupMessages(ctx context.Context, groupID uuid.UUID, limit int) ([]model.GroupMessage, error) {
	query := `
        SELECT id, group_id, sender_id, message_type, content, is_deleted, created_at, updated_at
        FROM group_messages
        WHERE group_id = $1 AND is_deleted = FALSE
        ORDER BY created_at DESC
        LIMIT $2
    `
	rows, err := api.Deps.DB.Pool().Query(ctx, query, groupID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying group messages: %w", err)
	}
	defer rows.Close()

	var messages []model.GroupMessage
	for rows.Next() {
		var msg model.GroupMessage
		err := rows.Scan(
			&msg.ID, &msg.GroupID, &msg.UserID, &msg.MessageType,
			&msg.Content, &msg.IsDeleted, &msg.CreatedAt, &msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning group message: %w", err)
		}
		messages = append(messages, msg)
	}

	// Reverse so oldest is first, if preferred by client
	// Or leave DESC so newest is first.
	return messages, nil
}

func (api *API) LeaveCommunityGroup(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	query := `
        DELETE FROM group_memberships
        WHERE group_id = $1 AND user_id = $2
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, query, groupID, userID)
	return err
}

func (api *API) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]model.GroupMembership, error) {
	query := `
        SELECT id, group_id, user_id, role, 'active' AS status, joined_at, updated_at
        FROM group_memberships
        WHERE group_id = $1
    `
	rows, err := api.Deps.DB.Pool().Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("querying group members: %w", err)
	}
	defer rows.Close()

	var members []model.GroupMembership
	for rows.Next() {
		var m model.GroupMembership
		err := rows.Scan(
			&m.ID, &m.GroupID, &m.UserID, &m.Role, &m.Status, &m.JoinedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning group member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (api *API) InsertGroupMessage(ctx context.Context, message model.GroupMessage) (model.GroupMessage, error) {
	message.ID = uuid.New()
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	query := `
        INSERT INTO group_messages (id, group_id, sender_id, message_type, content, is_deleted, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at, updated_at
    `
	err := api.Deps.DB.Pool().QueryRow(ctx, query,
		message.ID, message.GroupID, message.UserID, message.MessageType,
		message.Content, message.IsDeleted, message.CreatedAt, message.UpdatedAt,
	).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)

	if err != nil {
		return message, fmt.Errorf("inserting group message: %w", err)
	}

	// Also update last_message_at in the group
	_, _ = api.Deps.DB.Pool().Exec(ctx, `
        UPDATE community_groups SET last_message_at = $1 WHERE id = $2
    `, message.CreatedAt, message.GroupID)

	return message, nil
}
