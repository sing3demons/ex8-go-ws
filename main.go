package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection represents a WebSocket connection with metadata
type Connection struct {
	ID       string
	Conn     *websocket.Conn
	LastSeen time.Time
	Send     chan []byte // Channel สำหรับส่งข้อความ
}

// Message represents a message to be broadcasted
type Message struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Timestamp time.Time `json:"timestamp"`
}

// BroadcastMessage represents a message with exclusion info
type BroadcastMessage struct {
	Message   *Message
	ExcludeID string // ID ของ connection ที่ไม่ต้องการส่งไป
}

// ConnectionManager manages all WebSocket connections
type ConnectionManager struct {
	connections map[string]*Connection
	mutex       sync.RWMutex
	broadcast   chan *BroadcastMessage
	register    chan *Connection
	unregister  chan *Connection
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
		broadcast:   make(chan *BroadcastMessage, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
	}
}

// Run starts the connection manager's main loop
func (cm *ConnectionManager) Run() {
	for {
		select {
		case conn := <-cm.register:
			cm.registerConnection(conn)

		case conn := <-cm.unregister:
			cm.unregisterConnection(conn)

		case broadcastMsg := <-cm.broadcast:
			cm.broadcastMessage(broadcastMsg)
		}
	}
}

// registerConnection adds a new connection
func (cm *ConnectionManager) registerConnection(conn *Connection) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.connections[conn.ID] = conn
	log.Printf("📝 Connection registered: %s (Total: %d)", conn.ID, len(cm.connections))

	// ส่งข้อความต้อนรับ
	welcomeMsg := &Message{
		Type:      "system",
		Content:   "ยินดีต้อนรับสู่ระบบแชท! 🎉",
		Sender:    "System",
		Timestamp: time.Now(),
	}

	select {
	case conn.Send <- []byte(welcomeMsg.Content):
	default:
		close(conn.Send)
		delete(cm.connections, conn.ID)
	}
}

// unregisterConnection removes a connection
func (cm *ConnectionManager) unregisterConnection(conn *Connection) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.connections[conn.ID]; exists {
		delete(cm.connections, conn.ID)
		close(conn.Send)
		log.Printf("🗑️ Connection unregistered: %s (Total: %d)", conn.ID, len(cm.connections))
	}
}

// broadcastMessage sends a message to all connections except the sender
func (cm *ConnectionManager) broadcastMessage(broadcastMsg *BroadcastMessage) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	message := broadcastMsg.Message.Content
	excludeID := broadcastMsg.ExcludeID
	sentCount := 0

	for connID, conn := range cm.connections {
		// ไม่ส่งข้อความกลับไปยังผู้ส่ง
		if connID == excludeID {
			continue
		}

		select {
		case conn.Send <- []byte(message):
			sentCount++
		default:
			// Connection ไม่ตอบสนอง ลบออก
			close(conn.Send)
			delete(cm.connections, connID)
			log.Printf("🔌 Removed unresponsive connection: %s", connID)
		}
	}

	log.Printf("📡 Broadcasted message to %d connections (excluded: %s)", sentCount, excludeID)
}

// AddConnection adds a new connection to the manager
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn) string {
	// สร้าง unique ID สำหรับ connection
	connID := generateConnectionID()
	
	connection := &Connection{
		ID:       connID,
		Conn:     conn,
		LastSeen: time.Now(),
		Send:     make(chan []byte, 256),
	}

	cm.register <- connection
	return connID
}

// RemoveConnection removes a connection from the manager
func (cm *ConnectionManager) RemoveConnection(connID string) {
	cm.mutex.RLock()
	conn, exists := cm.connections[connID]
	cm.mutex.RUnlock()

	if exists {
		cm.unregister <- conn
	}
}

// BroadcastMessage broadcasts a message to all connections except sender
func (cm *ConnectionManager) BroadcastMessage(message *Message, excludeID string) {
	broadcastMsg := &BroadcastMessage{
		Message:   message,
		ExcludeID: excludeID,
	}

	select {
	case cm.broadcast <- broadcastMsg:
	default:
		log.Println("⚠️ Broadcast channel is full, dropping message")
	}
}

// GetConnectionCount returns the number of active connections
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.connections)
}

// GetConnection returns a connection by ID
func (cm *ConnectionManager) GetConnection(connID string) (*Connection, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	conn, exists := cm.connections[connID]
	return conn, exists
}

// generateConnectionID creates a unique connection ID
func generateConnectionID() string {
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

// WebSocket upgrader สำหรับ upgrade HTTP connection เป็น WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// อนุญาตให้ทุก origin เชื่อมต่อได้ (สำหรับการพัฒนา)
		return true
	},
}

// Global connection manager
var connectionManager *ConnectionManager

// handleWebSocket จัดการ WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection เป็น WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// เพิ่ม connection ไปยัง manager
	connID := connectionManager.AddConnection(conn)
	clientAddr := conn.RemoteAddr().String()
	log.Printf("🔗 New WebSocket connection: %s (ID: %s)", clientAddr, connID)

	// เริ่ม goroutines สำหรับ read และ write
	go handleRead(conn, connID, clientAddr)
	go handleWrite(conn, connID, clientAddr)
}

// handleRead จัดการการอ่านข้อความจาก client
func handleRead(conn *websocket.Conn, connID, clientAddr string) {
	defer func() {
		connectionManager.RemoveConnection(connID)
		conn.Close()
		log.Printf("🔌 Connection closed: %s (ID: %s)", clientAddr, connID)
	}()

	// ตั้งค่า read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// อ่านข้อความจาก client
		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error from %s: %v", clientAddr, err)
			}
			break
		}

		messageContent := string(rawMessage)
		log.Printf("📨 Received from %s: %s", clientAddr, messageContent)

		// สร้าง message object
		message := &Message{
			Type:      "text",
			Content:   messageContent,
			Sender:    clientAddr, // ใช้ client address เป็น sender ชั่วคราว
			Timestamp: time.Now(),
		}

		// Broadcast ข้อความไปยัง clients อื่น (ไม่รวมผู้ส่ง)
		connectionManager.BroadcastMessage(message, connID)
	}
}

// handleWrite จัดการการเขียนข้อความไปยัง client
func handleWrite(conn *websocket.Conn, connID, clientAddr string) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	// ดึง connection object จาก manager
	connection, exists := connectionManager.GetConnection(connID)
	if !exists {
		log.Printf("❌ Connection not found: %s", connID)
		return
	}

	for {
		select {
		case message, ok := <-connection.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel ถูกปิด
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// ส่งข้อความไปยัง client
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Failed to send message to %s: %v", clientAddr, err)
				return
			}

		case <-ticker.C:
			// ส่ง ping เพื่อ keep connection alive
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Failed to send ping to %s: %v", clientAddr, err)
				return
			}
		}
	}
}

func main() {
	// สร้าง connection manager
	connectionManager = NewConnectionManager()

	// เริ่ม connection manager ใน goroutine
	go connectionManager.Run()

	// ตั้งค่า HTTP routes
	http.HandleFunc("/ws", handleWebSocket)

	// เสิร์ฟ static files สำหรับ test client
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	// เริ่มต้น server
	port := ":9090"
	log.Printf("🚀 Starting WebSocket Chat Server on port %s", port)
	log.Printf("📡 WebSocket endpoint: ws://localhost%s/ws", port)
	log.Printf("🌐 Test page: http://localhost%s", port)
	log.Printf("👥 Connection Manager: Ready")

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
