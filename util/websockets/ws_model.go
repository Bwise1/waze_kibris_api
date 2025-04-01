package websockets

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MsgTypeSubscribe     = "subscribe"
	MsgTypeReportUpdate  = "report_update"
	MsgTypeDirectMessage = "direct_message"
	MsgTypeVoteUpdate    = "vote_update"
	MsgTypeCommentUpdate = "comment_update"
)

// Client represents a connected WebSocket user
type Client struct {
	Conn      *websocket.Conn
	UserID    string
	Latitude  float64
	Longitude float64
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
	Type      string  `json:"type"`
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Content   string  `json:"content,omitempty"`
	Receiver  string  `json:"receiver,omitempty"`
}
