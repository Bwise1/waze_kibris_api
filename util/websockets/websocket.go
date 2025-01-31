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
	"net/http"

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

// NewWebSocketManager initializes a WebSocketManager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*websocket.Conn]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *websocket.Conn),
		send:       make(chan DirectMessage),
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

		case conn := <-manager.unregister:
			manager.mu.Lock()
			if client, exists := manager.clients[conn]; exists {
				delete(manager.clients, conn)
				conn.Close()
				log.Printf("Client %s disconnected", client.UserID)
			}
			manager.mu.Unlock()

		case message := <-manager.broadcast:
			manager.mu.Lock()
			for _, client := range manager.clients {
				if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
					client.Conn.Close()
					delete(manager.clients, client.Conn)
				}
			}
			manager.mu.Unlock()

		case direct := <-manager.send:
			manager.mu.Lock()
			for _, client := range manager.clients {
				if client.UserID == direct.ReceiverID {
					if err := client.Conn.WriteMessage(websocket.TextMessage, []byte(direct.Message)); err != nil {
						client.Conn.Close()
						delete(manager.clients, client.Conn)
					}
					break
				}
			}
			manager.mu.Unlock()
		}
	}
}

// HandleConnections upgrades HTTP requests to WebSocket connections
func (manager *WebSocketManager) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}

	client := &Client{Conn: conn}
	manager.register <- client

	defer func() {
		manager.unregister <- conn
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			manager.unregister <- conn
			break
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Invalid JSON:", err)
			continue
		}

		switch message.Type {
		case MsgTypeSubscribe:
			client.UserID = message.UserID
			client.Latitude = message.Latitude
			client.Longitude = message.Longitude

		case MsgTypeReportUpdate:
			manager.broadcast <- msg

		case MsgTypeDirectMessage:
			directMsg := DirectMessage{
				ReceiverID: message.Receiver,
				Message:    message.Content,
			}
			manager.send <- directMsg
		}
	}
}

// BroadcastReportUpdate sends reports only to nearby users
func (manager *WebSocketManager) BroadcastReportUpdate(report []byte, reportLat, reportLon float64, radius float64) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for _, client := range manager.clients {
		if isNearby(client.Latitude, client.Longitude, reportLat, reportLon, radius) {
			client.Conn.WriteMessage(websocket.TextMessage, report)
		}
	}
}

// isNearby checks if a user is within a given radius
func isNearby(userLat, userLon, reportLat, reportLon, radius float64) bool {
	// Simple distance formula (can be replaced with Haversine formula)
	return (userLat-reportLat)*(userLat-reportLat)+(userLon-reportLon)*(userLon-reportLon) <= (radius * radius)
}
