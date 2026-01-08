package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	rooms  map[string]bool
	mu     sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	userMap    map[string][]*Client        // userID -> clients
	rooms      map[string]map[*Client]bool // room -> clients
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	Target  string // "all", "user:<id>", "room:<name>"
	Type    string
	Payload interface{}
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		userMap:    make(map[string][]*Client),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the Hub main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.userID != "" {
				h.userMap[client.userID] = append(h.userMap[client.userID], client)
			}
			h.mu.Unlock()
			log.Printf("🔌 Client connected: %s", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from userMap
				if clients, ok := h.userMap[client.userID]; ok {
					for i, c := range clients {
						if c == client {
							h.userMap[client.userID] = append(clients[:i], clients[i+1:]...)
							break
						}
					}
				}

				// Remove from rooms
				for room := range client.rooms {
					if roomClients, ok := h.rooms[room]; ok {
						delete(roomClients, client)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("🔌 Client disconnected: %s", client.userID)

		case msg := <-h.broadcast:
			h.handleBroadcast(msg)
		}
	}
}

func (h *Hub) handleBroadcast(msg *BroadcastMessage) {
	data, err := json.Marshal(Message{
		Type:    msg.Type,
		Payload: msg.Payload,
	})
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	switch {
	case msg.Target == "all":
		for client := range h.clients {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	case len(msg.Target) > 5 && msg.Target[:5] == "user:":
		userID := msg.Target[5:]
		if clients, ok := h.userMap[userID]; ok {
			for _, client := range clients {
				select {
				case client.send <- data:
				default:
					close(client.send)
				}
			}
		}
	case len(msg.Target) > 5 && msg.Target[:5] == "room:":
		roomName := msg.Target[5:]
		if roomClients, ok := h.rooms[roomName]; ok {
			for client := range roomClients {
				select {
				case client.send <- data:
				default:
					close(client.send)
				}
			}
		}
	}
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(msgType string, payload interface{}) {
	h.broadcast <- &BroadcastMessage{
		Target:  "all",
		Type:    msgType,
		Payload: payload,
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID, msgType string, payload interface{}) {
	h.broadcast <- &BroadcastMessage{
		Target:  "user:" + userID,
		Type:    msgType,
		Payload: payload,
	}
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(room, msgType string, payload interface{}) {
	h.broadcast <- &BroadcastMessage{
		Target:  "room:" + room,
		Type:    msgType,
		Payload: payload,
	}
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true
	client.rooms[room] = true
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomClients, ok := h.rooms[room]; ok {
		delete(roomClients, client)
	}
	delete(client.rooms, room)
}

// GetOnlineUsers returns list of online user IDs
func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.userMap))
	for userID := range h.userMap {
		users = append(users, userID)
	}
	return users
}

// ServeWs handles websocket requests from the peer
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, jwtSecret string) {
	// Extract token from query
	token := r.URL.Query().Get("token")
	userID := ""

	if token != "" {
		claims := &jwt.RegisteredClaims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err == nil && parsedToken.Valid {
			userID = claims.Subject
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		rooms:  make(map[string]bool),
	}

	hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "ping":
		c.send <- []byte(`{"type":"pong"}`)

	case "join_room":
		if room, ok := msg.Payload.(string); ok {
			c.hub.JoinRoom(c, room)
		}

	case "leave_room":
		if room, ok := msg.Payload.(string); ok {
			c.hub.LeaveRoom(c, room)
		}

	case "subscribe":
		// Subscribe to specific events
		if events, ok := msg.Payload.([]interface{}); ok {
			for _, event := range events {
				if eventName, ok := event.(string); ok {
					c.hub.JoinRoom(c, "event:"+eventName)
				}
			}
		}
	}
}

// TerminalMessage represents a terminal log message
type TerminalMessage struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"level"`  // info, warn, error, success, debug
	Source    string      `json:"source"` // wallet, account, campaign, task, browser, system
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	WalletID  string      `json:"wallet_id,omitempty"`
	AccountID string      `json:"account_id,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
}

// BroadcastTerminal sends a terminal message to a user
func (h *Hub) BroadcastTerminal(userID string, msg TerminalMessage) {
	msg.Timestamp = time.Now()
	h.BroadcastToUser(userID, "terminal", msg)
}

// TaskStatusUpdate represents a task status change
type TaskStatusUpdate struct {
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	Progress       int    `json:"progress"`
	Message        string `json:"message"`
	RequiresManual bool   `json:"requires_manual"`
	BrowserURL     string `json:"browser_url,omitempty"`
}

// BroadcastTaskUpdate sends task status update to a user
func (h *Hub) BroadcastTaskUpdate(userID string, update TaskStatusUpdate) {
	h.BroadcastToUser(userID, "task:status", update)
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Extract user ID from query param or JWT
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		rooms:  make(map[string]bool),
	}

	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}
