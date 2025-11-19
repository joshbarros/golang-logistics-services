package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	// ErrClientNotFound is returned when client is not found
	ErrClientNotFound = errors.New("client not found")
	// ErrInvalidMessage is returned when message is invalid
	ErrInvalidMessage = errors.New("invalid message")
)

// Message represents a WebSocket message
type Message struct {
	// Type identifies the message type
	Type string `json:"type"`
	// Data contains the message payload
	Data interface{} `json:"data"`
	// Timestamp when the message was created
	Timestamp time.Time `json:"timestamp"`
	// From identifies the sender (client ID or server)
	From string `json:"from,omitempty"`
	// To identifies the recipient (for targeted messages)
	To string `json:"to,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID         string
	conn       *websocket.Conn
	hub        *Hub
	send       chan Message
	UserID     string
	Metadata   map[string]string
	closeChan  chan struct{}
	closeOnce  sync.Once
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients       map[string]*Client
	clientsByUser map[string][]*Client
	broadcast     chan Message
	register      chan *Client
	unregister    chan *Client
	mutex         sync.RWMutex
	onConnect     func(client *Client)
	onDisconnect  func(client *Client)
	onMessage     func(client *Client, message Message) error
}

// Config holds WebSocket configuration
type Config struct {
	// ReadBufferSize for WebSocket connections
	ReadBufferSize int
	// WriteBufferSize for WebSocket connections
	WriteBufferSize int
	// MaxMessageSize in bytes
	MaxMessageSize int64
	// PongWait is how long to wait for pong response
	PongWait time.Duration
	// PingPeriod for sending pings
	PingPeriod time.Duration
	// WriteWait is timeout for write operations
	WriteWait time.Duration
	// OnConnect callback when client connects
	OnConnect func(client *Client)
	// OnDisconnect callback when client disconnects
	OnDisconnect func(client *Client)
	// OnMessage callback for incoming messages
	OnMessage func(client *Client, message Message) error
}

// DefaultConfig returns default WebSocket configuration
func DefaultConfig() Config {
	return Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		MaxMessageSize:  512 * 1024, // 512 KB
		PongWait:        60 * time.Second,
		PingPeriod:      54 * time.Second,
		WriteWait:       10 * time.Second,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// NewHub creates a new WebSocket hub
func NewHub(config Config) *Hub {
	return &Hub{
		clients:       make(map[string]*Client),
		clientsByUser: make(map[string][]*Client),
		broadcast:     make(chan Message, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		onConnect:     config.OnConnect,
		onDisconnect:  config.OnDisconnect,
		onMessage:     config.OnMessage,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client.ID] = client

	if client.UserID != "" {
		h.clientsByUser[client.UserID] = append(h.clientsByUser[client.UserID], client)
	}

	if h.onConnect != nil {
		h.onConnect(client)
	}
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		delete(h.clients, client.ID)
		close(client.send)

		if client.UserID != "" {
			// Remove from user's client list
			clients := h.clientsByUser[client.UserID]
			for i, c := range clients {
				if c.ID == client.ID {
					h.clientsByUser[client.UserID] = append(clients[:i], clients[i+1:]...)
					break
				}
			}

			// Clean up if no more clients for user
			if len(h.clientsByUser[client.UserID]) == 0 {
				delete(h.clientsByUser, client.UserID)
			}
		}

		if h.onDisconnect != nil {
			h.onDisconnect(client)
		}
	}
}

// broadcastMessage broadcasts a message to all clients
func (h *Hub) broadcastMessage(message Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// If targeted message, send only to specific client
	if message.To != "" {
		if client, exists := h.clients[message.To]; exists {
			select {
			case client.send <- message:
			default:
				// Client buffer full, skip
			}
		}
		return
	}

	// Broadcast to all clients
	for _, client := range h.clients {
		select {
		case client.send <- message:
		default:
			// Client buffer full, skip
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message Message) {
	h.broadcast <- message
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, message Message) error {
	h.mutex.RLock()
	client, exists := h.clients[clientID]
	h.mutex.RUnlock()

	if !exists {
		return ErrClientNotFound
	}

	message.To = clientID

	select {
	case client.send <- message:
		return nil
	default:
		return errors.New("client send buffer full")
	}
}

// SendToUser sends a message to all connections for a user
func (h *Hub) SendToUser(userID string, message Message) error {
	h.mutex.RLock()
	clients, exists := h.clientsByUser[userID]
	h.mutex.RUnlock()

	if !exists || len(clients) == 0 {
		return ErrClientNotFound
	}

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			// Skip if buffer full
		}
	}

	return nil
}

// GetClient returns a client by ID
func (h *Hub) GetClient(clientID string) (*Client, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	client, exists := h.clients[clientID]
	if !exists {
		return nil, ErrClientNotFound
	}

	return client, nil
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return len(h.clients)
}

// GetUserClients returns all clients for a user
func (h *Hub) GetUserClients(userID string) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.clientsByUser[userID]
}

// HandleWebSocket handles WebSocket upgrade and connection
func HandleWebSocket(hub *Hub, config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user info from context (set by auth middleware)
		userID, _ := c.Get("user_id")

		// Upgrade connection
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
			return
		}

		// Create client
		client := &Client{
			ID:        generateClientID(),
			conn:      conn,
			hub:       hub,
			send:      make(chan Message, 256),
			UserID:    fmt.Sprintf("%v", userID),
			Metadata:  make(map[string]string),
			closeChan: make(chan struct{}),
		}

		// Set connection parameters
		conn.SetReadLimit(config.MaxMessageSize)
		conn.SetReadDeadline(time.Now().Add(config.PongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(config.PongWait))
			return nil
		})

		// Register client
		hub.register <- client

		// Start goroutines for read and write
		go client.writePump(config)
		go client.readPump(config, hub)
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(config Config, hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {
		select {
		case <-c.closeChan:
			return
		default:
			var message Message
			err := c.conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket error: %v\n", err)
				}
				return
			}

			// Set message metadata
			message.From = c.ID
			message.Timestamp = time.Now()

			// Handle message
			if hub.onMessage != nil {
				if err := hub.onMessage(c, message); err != nil {
					fmt.Printf("Error handling message: %v\n", err)
				}
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump(config Config) {
	ticker := time.NewTicker(config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(config.WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(config.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.closeChan:
			return
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(message Message) error {
	select {
	case c.send <- message:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// Common message types
const (
	MessageTypeOrderUpdate      = "order.update"
	MessageTypeShipmentUpdate   = "shipment.update"
	MessageTypeDriverLocation   = "driver.location"
	MessageTypeNotification     = "notification"
	MessageTypeChat             = "chat"
	MessageTypeSystemAlert      = "system.alert"
)

// Example usage:
//
// // Create WebSocket hub
// wsHub := websocket.NewHub(websocket.Config{
//     ReadBufferSize:  1024,
//     WriteBufferSize: 1024,
//     OnConnect: func(client *websocket.Client) {
//         log.Printf("Client connected: %s (user: %s)", client.ID, client.UserID)
//     },
//     OnDisconnect: func(client *websocket.Client) {
//         log.Printf("Client disconnected: %s", client.ID)
//     },
//     OnMessage: func(client *websocket.Client, message websocket.Message) error {
//         log.Printf("Received message from %s: %+v", client.ID, message)
//         return nil
//     },
// })
//
// // Start hub
// go wsHub.Run()
//
// // Setup WebSocket endpoint
// router.GET("/ws", websocket.HandleWebSocket(wsHub, websocket.DefaultConfig()))
//
// // Broadcast updates
// go func() {
//     for event := range eventChannel {
//         message := websocket.Message{
//             Type: websocket.MessageTypeOrderUpdate,
//             Data: event,
//             Timestamp: time.Now(),
//         }
//         wsHub.Broadcast(message)
//     }
// }()
//
// // Send to specific user
// wsHub.SendToUser("user-123", websocket.Message{
//     Type: websocket.MessageTypeNotification,
//     Data: map[string]string{
//         "title": "Order Delivered",
//         "body":  "Your order has been delivered",
//     },
//     Timestamp: time.Now(),
// })
