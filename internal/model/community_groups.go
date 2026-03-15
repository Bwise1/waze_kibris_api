package model

import (
	"time"

	"github.com/google/uuid"
)

type CommunityGroup struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	ShortCode           string     `json:"short_code"`
	Description         *string    `json:"description"`
	GroupType           string     `json:"group_type"`
	DestinationPlaceID  *string    `json:"destination_place_id,omitempty"`
	DestinationName     *string    `json:"destination_name,omitempty"`
	DestinationLocation *string    `json:"destination_location,omitempty"` // WKT format for geometry
	Visibility          string     `json:"visibility"`
	CreatorID           uuid.UUID  `json:"creator_id,omitempty"`
	IconURL             *string    `json:"icon_url,omitempty"`
	MemberCount         int        `json:"member_count"`
	IsMember            bool       `json:"is_member"` // true if the current user is in this group
	LastMessageAt       *time.Time `json:"last_message_at,omitempty"`
	IsDeleted           bool       `json:"is_deleted"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type GroupMembership struct {
	ID        uuid.UUID  `json:"id"`
	GroupID   uuid.UUID  `json:"group_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Role      string     `json:"role"`   // "admin" or "member"
	Status    string     `json:"status"` // "active", "pending", or "invited"
	JoinedAt  time.Time  `json:"joined_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	IsDeleted bool       `json:"is_deleted"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type GroupMessage struct {
	ID             uuid.UUID  `json:"id"`
	GroupID        uuid.UUID  `json:"group_id"`
	UserID         uuid.UUID  `json:"user_id"`
	SenderUsername *string    `json:"sender_username,omitempty"` // from JOIN with users, for display
	MessageType    string     `json:"message_type"`             // "text", "location", "system"
	Content        string     `json:"content"`
	IsDeleted      bool       `json:"is_deleted"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// GroupInvitation represents an invite to join a community group.
type GroupInvitation struct {
	ID             uuid.UUID  `json:"id"`
	GroupID        uuid.UUID  `json:"group_id"`
	InvitedUserID  uuid.UUID  `json:"invited_user_id"`
	InvitedBy      *uuid.UUID `json:"invited_by,omitempty"`
	Status         string     `json:"status"` // "pending", "accepted", "declined", "revoked"
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// Optional joined fields for list responses
	GroupName      *string `json:"group_name,omitempty"`
	InvitedByName  *string `json:"invited_by_name,omitempty"`
	InvitedUserEmail *string `json:"invited_user_email,omitempty"`
}
