package model

import (
	"time"

	"github.com/google/uuid"
)

type Vote struct {
	ID        uuid.UUID `json:"id"`
	ReportID  int64     `json:"report_id"`
	UserID    uuid.UUID `json:"user_id"`
	VoteType  string    `json:"vote_type"`
	CreatedAt time.Time `json:"created_at"`
}
