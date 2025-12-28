package websocket

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"realtime-chat/internal/config"
)

// Connection interface for WebSocket connections (to avoid import cycle)
type Connection interface {
	GetID() string
	GetUser() interface{}
	SetUser(user interface{})
	SendMessage(message []byte) error
	GetSendChannel() chan []byte
	IsHealthy(timeout time.Duration) bool
	GetHealthStats() *config.ConnectionHealth
	Close() error
}

// Message represents a message to be broadcasted (to avoid import cycle)
type Message struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// BroadcastMessage represents a message with exclusion info (to avoid import cycle)
type BroadcastMessage struct {
	Message   *Message
	ExcludeID string // ID ของ connection ที่ไม่ต้องการส่งไป
	RoomName  string // ชื่อห้องที่จะส่งข้อความ (ถ้าว่างจะส่งให้ทุกคน)
}

// UserService interface (to avoid import cycle)
type UserService interface {
	UnregisterUser(connID string) error
}

// RoomService interface (to avoid import cycle)
type RoomService interface {
	LeaveRoom(user interface{}, roomName string) error
}

// UserInterface defines the interface for user objects (to avoid import cycle)
type UserInterface interface {
	GetIsAuthenticated() bool
	GetUsername() string
	GetCurrentRoom() string
}

// MessageInterface defines the interface for message objects (to avoid import cycle)
type MessageInterface interface {
	GetType() string
	GetContent() string
	GetSender() string
	GetUsername() string
	GetTimestamp() time.Time
}

// Manager manages WebSocket connections and message broadcasting
type Manager struct {
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
	broadcast   chan *BroadcastMessage
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	config      *config.ServerConfig
	userService UserService
	roomService RoomService
	metrics     *config.ServerMetrics
}

// NewManager creates a new WebSocket manager
func NewManager(cfg *config.ServerConfig, userService UserService, roomService RoomService, metrics *config.ServerMetrics) *Manager {
	return &Manager{
		connections: make(map[string]*WebSocketConnection),
		broadcast:   make(chan *BroadcastMessage, cfg.BroadcastBuffer),
		register:    make(chan *WebSocketConnection),
		unregister:  make(chan *WebSocketConnection),
		config:      cfg,
		userService: userService,
		roomService: roomService,
		metrics:     metrics,
	}
}

// Run starts the manager's main loop
func (m *Manager) Run() {
	// เริ่ม health check goroutine ถ้า enable
	if m.config.EnableHealthCheck {
		go m.runHealthCheck()
	}
	
	for {
		select {
		case conn := <-m.register:
			m.registerConnection(conn)

		case conn := <-m.unregister:
			m.unregisterConnection(conn)

		case broadcastMsg := <-m.broadcast:
			m.broadcastMessage(broadcastMsg)
		}
	}
}

// AddConnection adds a new WebSocket connection
func (m *Manager) AddConnection(conn *websocket.Conn) string {
	connID := GenerateConnectionID()
	
	wsConn := NewWebSocketConnection(connID, conn)
	m.register <- wsConn
	
	return connID
}

// RemoveConnection removes a connection
func (m *Manager) RemoveConnection(connID string) {
	m.mutex.RLock()
	conn, exists := m.connections[connID]
	m.mutex.RUnlock()

	if exists {
		m.unregister <- conn
	}
}

// GetConnection returns a connection by ID
func (m *Manager) GetConnection(connID string) (Connection, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	conn, exists := m.connections[connID]
	return conn, exists
}

// GetConnectionCount returns the number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.connections)
}

// BroadcastMessage broadcasts a message to all connections except sender (adapter for interface compatibility)
func (m *Manager) BroadcastMessage(message interface{}, excludeID string) {
	// Convert interface{} message to our internal Message type
	if msg, ok := message.(*Message); ok {
		m.BroadcastToRoom(msg, excludeID, "")
	} else {
		// Try to convert from chat.Message type
		if chatMsg, ok := message.(MessageInterface); ok {
			msg := &Message{
				Type:      chatMsg.GetType(),
				Content:   chatMsg.GetContent(),
				Sender:    chatMsg.GetSender(),
				Username:  chatMsg.GetUsername(),
				Timestamp: chatMsg.GetTimestamp(),
			}
			m.BroadcastToRoom(msg, excludeID, "")
		}
	}
}

// BroadcastToRoom broadcasts a message to connections in a specific room (adapter for interface compatibility)
func (m *Manager) BroadcastToRoom(message interface{}, excludeID, roomName string) {
	var msg *Message
	
	// Convert interface{} message to our internal Message type
	if m, ok := message.(*Message); ok {
		msg = m
	} else {
		// Try to convert from chat.Message type
		if chatMsg, ok := message.(MessageInterface); ok {
			msg = &Message{
				Type:      chatMsg.GetType(),
				Content:   chatMsg.GetContent(),
				Sender:    chatMsg.GetSender(),
				Username:  chatMsg.GetUsername(),
				Timestamp: chatMsg.GetTimestamp(),
			}
		} else {
			log.Println("⚠️ Unknown message type in BroadcastToRoom")
			return
		}
	}

	broadcastMsg := &BroadcastMessage{
		Message:   msg,
		ExcludeID: excludeID,
		RoomName:  roomName,
	}

	select {
	case m.broadcast <- broadcastMsg:
	default:
		log.Println("⚠️ Broadcast channel is full, dropping message")
	}
}

// registerConnection adds a new connection
func (m *Manager) registerConnection(conn *WebSocketConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// ตรวจสอบ connection limits
	if len(m.connections) >= m.config.MaxConnections {
		log.Printf("❌ Connection limit reached, rejecting: %s", conn.ID)
		conn.Conn.WriteMessage(websocket.TextMessage, []byte("❌ เซิร์ฟเวอร์เต็ม กรุณาลองใหม่ภายหลัง"))
		conn.Conn.Close()
		return
	}

	m.connections[conn.ID] = conn
	m.metrics.IncrementConnections()
	log.Printf("📝 Connection registered: %s (Total: %d/%d)", conn.ID, len(m.connections), m.config.MaxConnections)

	// ส่งข้อความขอ username
	authMsg := &Message{
		Type:      "auth_request",
		Content:   "กรุณาระบุชื่อผู้ใช้ของคุณ:",
		Sender:    "System",
		Username:  "System",
		Timestamp: time.Now(),
	}

	select {
	case conn.Send <- []byte(authMsg.Content):
	default:
		close(conn.Send)
		delete(m.connections, conn.ID)
	}
}

// unregisterConnection removes a connection
func (m *Manager) unregisterConnection(conn *WebSocketConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.connections[conn.ID]; exists {
		// ถ้ามี user ให้แจ้งเตือนคนอื่น
		if conn.User != nil {
			// Type assertion to access user fields
			if user, ok := conn.User.(UserInterface); ok && user.GetIsAuthenticated() {
				// ส่งข้อความแจ้งว่ามีคนออก
				leaveMsg := &Message{
					Type:      "user_left",
					Content:   fmt.Sprintf("👋 %s ออกจากระบบแล้ว", user.GetUsername()),
					Sender:    "System",
					Username:  "System",
					Timestamp: time.Now(),
				}
				
				// Broadcast ข้อความแจ้งให้คนอื่นรู้
				m.broadcastMessage(&BroadcastMessage{
					Message:   leaveMsg,
					ExcludeID: "", // ส่งให้ทุกคน
				})

				// ออกจากห้องปัจจุบัน
				if user.GetCurrentRoom() != "" {
					m.roomService.LeaveRoom(conn.User, user.GetCurrentRoom())
				}

				// ลบ user จาก user service
				m.userService.UnregisterUser(conn.ID)
				m.metrics.DecrementUsers()
			}
		}

		delete(m.connections, conn.ID)
		close(conn.Send)
		m.metrics.DecrementConnections()
		log.Printf("🗑️ Connection unregistered: %s (Total: %d/%d)", conn.ID, len(m.connections), m.config.MaxConnections)
	}
}

// broadcastMessage sends a message to all connections except the sender
func (m *Manager) broadcastMessage(broadcastMsg *BroadcastMessage) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	message := broadcastMsg.Message
	excludeID := broadcastMsg.ExcludeID
	roomName := broadcastMsg.RoomName
	sentCount := 0

	// สร้างข้อความที่จะส่ง
	var formattedMessage string
	if message.Type == "text" && message.Username != "" {
		formattedMessage = fmt.Sprintf("[%s]: %s", message.Username, message.Content)
	} else {
		formattedMessage = message.Content
	}

	for connID, conn := range m.connections {
		// ไม่ส่งข้อความกลับไปยังผู้ส่ง
		if connID == excludeID {
			continue
		}

		// ตรวจสอบว่า connection มี user และอยู่ในห้องที่ถูกต้องหรือไม่
		if roomName != "" && conn.User != nil {
			// Type assertion to access CurrentRoom field
			if user, ok := conn.User.(UserInterface); ok {
				if user.GetCurrentRoom() != roomName {
					continue // ไม่อยู่ในห้องเดียวกัน ข้าม
				}
			}
		}

		select {
		case conn.Send <- []byte(formattedMessage):
			sentCount++
		default:
			// Connection ไม่ตอบสนอง ลบออก
			close(conn.Send)
			delete(m.connections, connID)
			log.Printf("🔌 Removed unresponsive connection: %s", connID)
		}
	}

	// นับ message metrics
	if message.Type == "text" {
		m.metrics.IncrementMessages()
	}

	if roomName != "" {
		log.Printf("📡 Broadcasted message to %d connections in room '%s' (excluded: %s)", sentCount, roomName, excludeID)
	} else {
		log.Printf("📡 Broadcasted message to %d connections (excluded: %s)", sentCount, excludeID)
	}
}

// runHealthCheck runs periodic health checks on all connections
func (m *Manager) runHealthCheck() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	log.Printf("💓 Starting connection health monitor (interval: %v)", m.config.HealthCheckInterval)
	
	for {
		select {
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck checks health of all connections and removes unhealthy ones
func (m *Manager) performHealthCheck() {
	m.mutex.RLock()
	unhealthyConnections := make([]*WebSocketConnection, 0)
	healthyCount := 0
	
	for _, conn := range m.connections {
		if !conn.IsHealthy(m.config.PongTimeout) {
			unhealthyConnections = append(unhealthyConnections, conn)
		} else {
			healthyCount++
		}
	}
	m.mutex.RUnlock()
	
	// ลบ connections ที่ไม่ healthy
	for _, conn := range unhealthyConnections {
		log.Printf("💔 Removing unhealthy connection: %s (missed pongs: %d)", 
			conn.ID, conn.Health.GetStats().MissedPongs)
		m.unregister <- conn
	}
	
	if len(unhealthyConnections) > 0 {
		log.Printf("💓 Health check completed: %d healthy, %d removed", 
			healthyCount, len(unhealthyConnections))
	}
}

// GetConnectionHealth returns health statistics for a connection
func (m *Manager) GetConnectionHealth(connID string) (*config.ConnectionHealth, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if conn, exists := m.connections[connID]; exists {
		return conn.GetHealthStats(), true
	}
	return nil, false
}

// GetAllConnectionsHealth returns health statistics for all connections
func (m *Manager) GetAllConnectionsHealth() map[string]*config.ConnectionHealth {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	healthStats := make(map[string]*config.ConnectionHealth)
	for id, conn := range m.connections {
		healthStats[id] = conn.GetHealthStats()
	}
	return healthStats
}