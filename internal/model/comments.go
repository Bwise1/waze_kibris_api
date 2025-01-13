package model

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID `json:"id"`
	ReportID  uuid.UUID `json:"report_id"`
	UserID    uuid.UUID `json:"user_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}
