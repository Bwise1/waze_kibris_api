package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SavedLocation struct {
	ID        int64        `json:"id"`
	UserID    uuid.UUID    `json:"user_id"`
	Name      string       `json:"name"`
	Location  pgtype.Point `json:"location"`
	CreatedAt time.Time    `json:"created_at"`
}

type LocationRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=50"`
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

type SavedLocationResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
