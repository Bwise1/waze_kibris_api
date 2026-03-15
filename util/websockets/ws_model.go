package websockets

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MsgTypeSubscribe           = "subscribe"
	MsgTypeReportUpdate        = "report_update"
	MsgTypeDirectMessage       = "direct_message"
	MsgTypeVoteUpdate          = "vote_update"
	MsgTypeCommentUpdate       = "comment_update"
	MsgTypeGroupChat           = "group_chat"
	MsgTypeGroupLocationUpdate = "group_location_update"
)

// ReportUpdatePayload is sent in Message.Content for report_update events.
type ReportUpdatePayload struct {
	ID             int64   `json:"id"`
	UserID         string  `json:"user_id"`
	Type           string  `json:"type"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Active         bool    `json:"active"`
	Resolved       bool    `json:"resolved"`
	UpvotesCount   int     `json:"upvotes_count"`
	DownvotesCount int     `json:"downvotes_count"`
}

// Client represents a connected WebSocket user
type Client struct {
	Conn           *websocket.Conn
	UserID         string
	Latitude       float64
	Longitude      float64
	ActiveGroupIDs []string
}

type WebSocketManager struct {
	clients    map[*websocket.Conn]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *websocket.Conn
	send       chan DirectMessage
	mu         sync.Mutex
}

// DirectMessage struct for 1-on-1 messages
type DirectMessage struct {
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

// Message struct for incoming WebSocket messages
type Message struct {
	Type           string   `json:"type"`
	UserID         string   `json:"user_id"`
	Latitude       float64  `json:"latitude,omitempty"`
	Longitude      float64  `json:"longitude,omitempty"`
	Content        string   `json:"content,omitempty"`
	Receiver       string   `json:"receiver,omitempty"`
	GroupID        string   `json:"group_id,omitempty"`
	ActiveGroupIDs []string `json:"active_group_ids,omitempty"`
}
