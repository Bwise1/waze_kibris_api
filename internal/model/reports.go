package model

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Type           string    `json:"type"`              // TRAFFIC, POLICE, ACCIDENT, HAZARD, ROAD_CLOSED
	Subtype        string    `json:"subtype,omitempty"` // LIGHT, HEAVY, STAND_STILL, VISIBLE, HIDDEN, OTHER_SIDE, MINOR, MAJOR
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Description    string    `json:"description"`
	Severity       int       `json:"severity"`
	VerifiedCount  int       `json:"verified_count"`
	Active         bool      `json:"active"`
	Resolved       bool      `json:"resolved"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	ImageURL       string    `json:"image_url"`
	ReportSource   string    `json:"report_source"`
	ReportStatus   string    `json:"report_status"`
	CommentsCount  int       `json:"comments_count"`
	UpvotesCount   int       `json:"upvotes_count"`
	DownvotesCount int       `json:"downvotes_count"`
}
