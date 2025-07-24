package rest

import (
	"context"
	"fmt"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (api *API) CreateSavedLocationRepo(ctx context.Context, location model.SavedLocation) error {

	stmt := `
        INSERT INTO saved_locations (user_id, name, location, place_id)
        VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326), $5)
    `
	_, err := api.Deps.DB.Pool().Exec(ctx, stmt,
		location.UserID,
		location.Name,
		location.Location.P.X,
		location.Location.P.Y,
		location.PlaceID,
	)
	if err != nil {
		return fmt.Errorf("creating saved location: %w", err)
	}
	return nil
}

func (api *API) GetSavedLocationRepo(ctx context.Context, id int64) (model.SavedLocation, error) {
	var location model.SavedLocation
	stmt := `
        SELECT id, user_id, name,
               ST_X(location::geometry) as longitude,
               ST_Y(location::geometry) as latitude,
               place_id,
               created_at
        FROM saved_locations
        WHERE id = $1
    `

	err := api.Deps.DB.Pool().QueryRow(ctx, stmt, id).Scan(
		&location.ID,
		&location.UserID,
		&location.Name,
		&location.Location.P.X,
		&location.Location.P.Y,
		&location.PlaceID,
		&location.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.SavedLocation{}, nil
		}
		return model.SavedLocation{}, fmt.Errorf("getting saved location: %w", err)
	}

	return location, nil
}

func (api *API) UpdateSavedLocationRepo(ctx context.Context, location model.SavedLocation) error {
	stmt := `
        UPDATE public.saved_locations
        SET name = $2,
            location = ST_SetSRID(ST_MakePoint($3, $4), 4326),
            place_id = $5,
            updated_at = NOW()
        WHERE id = $1
    `
	result, err := api.Deps.DB.Pool().Exec(ctx, stmt,
		location.ID,
		location.Name,
		location.Location.P.X,
		location.Location.P.Y,
		location.PlaceID,
	)
	if err != nil {
		return fmt.Errorf("updating saved location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("location with ID %d not found", location.ID)
	}
	return nil
}

func (api *API) GetSavedLocationsRepo(ctx context.Context, userID uuid.UUID) ([]model.SavedLocationResponse, error) {
	stmt := `
		SELECT id, name, COALESCE(address, '') as address,
			   ST_X(location::geometry) as longitude,
			   ST_Y(location::geometry) as latitude,
			   place_id
		FROM saved_locations
		WHERE user_id = $1
	`
	rows, err := api.Deps.DB.Pool().Query(ctx, stmt, userID)
	if err != nil {
		return nil, fmt.Errorf("getting saved locations: %w", err)
	}
	defer rows.Close()

	var locations []model.SavedLocationResponse
	for rows.Next() {
		var location model.SavedLocationResponse
		err := rows.Scan(
			&location.ID,
			&location.Name,
			&location.Address,
			&location.Longitude,
			&location.Latitude,
			&location.PlaceID,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning saved location: %w", err)
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (api *API) DeleteSavedLocationRepo(ctx context.Context, id int64) error {
	stmt := `DELETE FROM public.saved_locations WHERE id = $1`

	result, err := api.Deps.DB.Pool().Exec(ctx, stmt, id)
	if err != nil {
		return fmt.Errorf("deleting saved location: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("location with ID %d not found", id)
	}
	return nil
}
