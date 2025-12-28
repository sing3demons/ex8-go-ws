package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"realtime-chat/internal/config"
)

// WebSocketConnection implements chat.Connection interface
type WebSocketConnection struct {
	ID       string
	Conn     *websocket.Conn
	User     interface{} // ใช้ interface{} เพื่อหลีกเลี่ยง import cycle
	LastSeen time.Time
	Send     chan []byte
	Health   *config.ConnectionHealth
}

// NewWebSocketConnection creates a new WebSocket connection
func NewWebSocketConnection(id string, conn *websocket.Conn) *WebSocketConnection {
	return &WebSocketConnection{
		ID:       id,
		Conn:     conn,
		LastSeen: time.Now(),
		Send:     make(chan []byte, 256),
		Health:   config.NewConnectionHealth(),
	}
}

// GetID returns the connection ID
func (c *WebSocketConnection) GetID() string {
	return c.ID
}

// GetUser returns the user associated with this connection
func (c *WebSocketConnection) GetUser() interface{} {
	return c.User
}

// SetUser sets the user for this connection
func (c *WebSocketConnection) SetUser(user interface{}) {
	c.User = user
}

// SendMessage sends a message through the connection
func (c *WebSocketConnection) SendMessage(message []byte) error {
	c.Health.RecordActivity()
	select {
	case c.Send <- message:
		return nil
	default:
		log.Printf("❌ Failed to send message to connection %s", c.ID)
		return nil
	}
}

// GetSendChannel returns the send channel for this connection
func (c *WebSocketConnection) GetSendChannel() chan []byte {
	return c.Send
}

// IsHealthy checks if the connection is healthy
func (c *WebSocketConnection) IsHealthy(pongTimeout time.Duration) bool {
	return c.Health.CheckHealth(pongTimeout)
}

// GetHealthStats returns connection health statistics
func (c *WebSocketConnection) GetHealthStats() *config.ConnectionHealth {
	return c.Health.GetStats()
}

// Close closes the connection
func (c *WebSocketConnection) Close() error {
	close(c.Send)
	return c.Conn.Close()
}

// generateConnectionID creates a unique connection ID
func GenerateConnectionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}