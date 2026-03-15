// package websockets

// import (
// 	"log"
// 	"net/http"

// 	"github.com/gorilla/websocket"
// )

// type WebSocketManager struct {
// 	clients    map[*websocket.Conn]bool
// 	broadcast  chan []byte
// 	register   chan *websocket.Conn
// 	unregister chan *websocket.Conn
// }

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

// func NewWebSocketManager() *WebSocketManager {
// 	return &WebSocketManager{
// 		clients:    make(map[*websocket.Conn]bool),
// 		broadcast:  make(chan []byte),
// 		register:   make(chan *websocket.Conn),
// 		unregister: make(chan *websocket.Conn),
// 	}
// }

// func (manager *WebSocketManager) Run() {
// 	for {
// 		select {
// 		case conn := <-manager.register:
// 			manager.clients[conn] = true
// 		case conn := <-manager.unregister:
// 			if _, ok := manager.clients[conn]; ok {
// 				delete(manager.clients, conn)
// 				conn.Close()
// 			}
// 		case message := <-manager.broadcast:
// 			for conn := range manager.clients {
// 				err := conn.WriteMessage(websocket.TextMessage, message)
// 				if err != nil {
// 					conn.Close()
// 					delete(manager.clients, conn)
// 				}
// 			}
// 		}
// 	}
// }

// func (manager *WebSocketManager) HandleConnections(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
// 	defer conn.Close()

// 	manager.register <- conn

// 	for {
// 		_, message, err := conn.ReadMessage()
// 		if err != nil {
// 			manager.unregister <- conn
// 			break
// 		}
// 		manager.broadcast <- message
// 	}
// }

package websockets

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager handles WebSocket connections and messaging
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	clientSendBufferSize = 256
	readLimit           = 512
	pongWait            = 60 * time.Second  // time to wait for pong before considering conn dead
	pingPeriod          = 30 * time.Second  // server sends ping this often
	writeWait           = 10 * time.Second  // deadline for write (ping or app message)
)

// NewWebSocketManager initializes a WebSocketManager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:      make(map[*websocket.Conn]*Client),
		userIndex:    make(map[string]*Client),
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		registerUser: make(chan *Client, 64),
		unregister:   make(chan *websocket.Conn),
		send:         make(chan DirectMessage),
	}
}

// writePump runs in a goroutine per client; it reads from client.Send and writes to the websocket.
// Sends a protocol-level ping every pingPeriod so the client responds with pong; readPump uses
// pong to extend the read deadline and detect dead connections.
// Exits when client.Send is closed (on unregister).
func (manager *WebSocketManager) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				return
			}
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("writePump error for client %s: %v", client.UserID, err)
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Printf("writePump ping error for client %s: %v", client.UserID, err)
				return
			}
		}
	}
}

// Run starts the WebSocket manager
func (manager *WebSocketManager) Run() {
	for {
		select {
		case client := <-manager.register:
			manager.mu.Lock()
			manager.clients[client.Conn] = client
			manager.mu.Unlock()
			go manager.writePump(client)

		case conn := <-manager.unregister:
			manager.mu.Lock()
			if client, exists := manager.clients[conn]; exists {
				delete(manager.clients, conn)
				if client.UserID != "" && manager.userIndex[client.UserID] == client {
					delete(manager.userIndex, client.UserID)
				}
				close(client.Send)
				log.Printf("Client %s disconnected", client.UserID)
			}
			manager.mu.Unlock()
			conn.Close()

		case client := <-manager.registerUser:
			manager.mu.Lock()
			manager.userIndex[client.UserID] = client
			manager.mu.Unlock()

		case message := <-manager.broadcast:
			manager.mu.Lock()
			clients := make([]*Client, 0, len(manager.clients))
			for _, c := range manager.clients {
				clients = append(clients, c)
			}
			manager.mu.Unlock()
			for _, client := range clients {
				select {
				case client.Send <- message:
				default:
					// buffer full; skip this client to avoid blocking
				}
			}

		case direct := <-manager.send:
			manager.mu.Lock()
			client := manager.userIndex[direct.ReceiverID]
			if client != nil {
				select {
				case client.Send <- []byte(direct.Message):
				default:
				}
			}
			manager.mu.Unlock()
		}
	}
}

// HandleConnections upgrades HTTP requests to WebSocket connections.
// The read loop (readPump) sets read limit, deadline, and pong handler so dead connections are detected.
func (manager *WebSocketManager) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}

	client := &Client{
		Conn: conn,
		Send: make(chan []byte, clientSendBufferSize),
	}
	manager.register <- client

	defer conn.Close()
	defer func() {
		manager.unregister <- conn
	}()

	// When client sends a close frame, unregister so Run() can clean up; return nil for default close response.
	conn.SetCloseHandler(func(code int, text string) error {
		manager.unregister <- conn
		return nil
	})

	// readPump: limit size, deadline, and pong handler to detect dead connections
	conn.SetReadLimit(readLimit)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(pongWait))

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Invalid JSON:", err)
			continue
		}

		switch message.Type {
		case "ping":
			// Keepalive from client; no reply needed, keeps connection alive past proxy timeouts

		case MsgTypeSubscribe:
			client.UserID = message.UserID
			client.Latitude = message.Latitude
			client.Longitude = message.Longitude
			if message.ActiveGroupIDs != nil {
				client.ActiveGroupIDs = message.ActiveGroupIDs
			}
			if client.UserID != "" {
				manager.registerUser <- client
			}

		case MsgTypeReportUpdate:
			manager.broadcast <- msg

		case MsgTypeDirectMessage:
			directMsg := DirectMessage{
				ReceiverID: message.Receiver,
				Message:    message.Content,
			}
			manager.send <- directMsg

		case MsgTypeGroupChat, MsgTypeGroupLocationUpdate:
			if message.GroupID != "" {
				manager.BroadcastToGroup(message.GroupID, msg)
			}
		}
	}
}

// BroadcastReportUpdate sends reports only to nearby users via each client's send channel
func (manager *WebSocketManager) BroadcastReportUpdate(report []byte, reportLat, reportLon float64, radius float64) {
	manager.mu.Lock()
	clients := make([]*Client, 0, len(manager.clients))
	for _, c := range manager.clients {
		if isNearby(c.Latitude, c.Longitude, reportLat, reportLon, radius) {
			clients = append(clients, c)
		}
	}
	manager.mu.Unlock()
	for _, client := range clients {
		select {
		case client.Send <- report:
		default:
		}
	}
}

// isNearby checks if a user is within a given radius using the Haversine formula
func isNearby(userLat, userLon, reportLat, reportLon, radius float64) bool {
	const earthRadius = 6371000 // Earth radius in meters

	lat1Rad := userLat * math.Pi / 180
	lat2Rad := reportLat * math.Pi / 180
	deltaLatRad := (reportLat - userLat) * math.Pi / 180
	deltaLonRad := (reportLon - userLon) * math.Pi / 180

	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLonRad/2)*math.Sin(deltaLonRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadius * c
	return distance <= radius
}

// GetNearbyUsers returns connected clients within radiusMeters of (lat, lon), excluding excludeUserID.
func (manager *WebSocketManager) GetNearbyUsers(lat, lon, radiusMeters float64, excludeUserID string) []NearbyUser {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	out := make([]NearbyUser, 0)
	for _, c := range manager.clients {
		if c.UserID == "" || c.UserID == excludeUserID {
			continue
		}
		if isNearby(c.Latitude, c.Longitude, lat, lon, radiusMeters) {
			out = append(out, NearbyUser{
				UserID:    c.UserID,
				Latitude:  c.Latitude,
				Longitude: c.Longitude,
			})
		}
	}
	return out
}

// BroadcastToGroup sends a message to all connected clients who have groupID in their ActiveGroupIDs
func (manager *WebSocketManager) BroadcastToGroup(groupID string, message []byte) {
	manager.mu.Lock()
	clients := make([]*Client, 0, len(manager.clients))
	for _, c := range manager.clients {
		for _, activeGrpID := range c.ActiveGroupIDs {
			if activeGrpID == groupID {
				clients = append(clients, c)
				break
			}
		}
	}
	manager.mu.Unlock()
	for _, client := range clients {
		select {
		case client.Send <- message:
		default:
		}
	}
}
