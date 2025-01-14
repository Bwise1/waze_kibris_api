package model

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID             int64     `json:"id"`
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

type CreateReportRequest struct {
	UserID       uuid.UUID `json:"user_id"`
	Type         string    `json:"type"`
	Subtype      *string   `json:"subtype,omitempty"`
	Longitude    float64   `json:"longitude"`
	Latitude     float64   `json:"latitude"`
	Description  *string   `json:"description,omitempty"`
	Severity     *int      `json:"severity,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	ImageURL     *string   `json:"image_url,omitempty"`
	ReportSource *string   `json:"report_source,omitempty"`
	ReportStatus *string   `json:"report_status,omitempty"`
}

type UpdateReportRequest struct {
	ID           int64     `json:"id" validate:"required"`
	Type         string    `json:"type" validate:"required"`
	Subtype      string    `json:"subtype,omitempty"`
	Latitude     float64   `json:"latitude" validate:"required"`
	Longitude    float64   `json:"longitude" validate:"required"`
	Description  string    `json:"description"`
	Severity     int       `json:"severity" validate:"required,min=1,max=5"`
	Active       bool      `json:"active"`
	Resolved     bool      `json:"resolved"`
	ExpiresAt    time.Time `json:"expires_at" validate:"required"`
	ImageURL     string    `json:"image_url"`
	ReportSource string    `json:"report_source" validate:"required"`
	ReportStatus string    `json:"report_status" validate:"required"`
}
