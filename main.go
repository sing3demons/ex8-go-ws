package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection represents a WebSocket connection with metadata
type Connection struct {
	ID       string
	Conn     *websocket.Conn
	User     *User // เพิ่ม User information
	LastSeen time.Time
	Send     chan []byte // Channel สำหรับส่งข้อความ
}

// User represents a chat user
type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	ConnID      string    `json:"conn_id"`
	CurrentRoom string    `json:"current_room"` // ห้องปัจจุบัน
	JoinedAt    time.Time `json:"joined_at"`
	LastActive  time.Time `json:"last_active"`
	IsAuthenticated bool  `json:"is_authenticated"`
}

// Room represents a chat room
type Room struct {
	Name        string            `json:"name"`
	Users       map[string]*User  `json:"users"`        // username -> User
	CreatedAt   time.Time         `json:"created_at"`
	CreatedBy   string            `json:"created_by"`
	MaxUsers    int               `json:"max_users"`
	IsActive    bool              `json:"is_active"`
}

// Message represents a message to be broadcasted
type Message struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Username  string    `json:"username"` // เพิ่ม username
	Timestamp time.Time `json:"timestamp"`
}

// BroadcastMessage represents a message with exclusion info
type BroadcastMessage struct {
	Message   *Message
	ExcludeID string // ID ของ connection ที่ไม่ต้องการส่งไป
	RoomName  string // ชื่อห้องที่จะส่งข้อความ (ถ้าว่างจะส่งให้ทุกคน)
}

// RoomManager manages chat rooms
type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	rm := &RoomManager{
		rooms: make(map[string]*Room),
	}
	
	// สร้างห้อง default
	defaultRoom := &Room{
		Name:      "general",
		Users:     make(map[string]*User),
		CreatedAt: time.Now(),
		CreatedBy: "System",
		MaxUsers:  100,
		IsActive:  true,
	}
	rm.rooms["general"] = defaultRoom
	
	log.Printf("🏠 Default room 'general' created")
	return rm
}

// CreateRoom creates a new room
func (rm *RoomManager) CreateRoom(name, creatorUsername string) (*Room, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// ตรวจสอบว่าห้องมีอยู่แล้วหรือไม่
	if _, exists := rm.rooms[name]; exists {
		return nil, fmt.Errorf("room '%s' already exists", name)
	}

	// สร้างห้องใหม่
	room := &Room{
		Name:      name,
		Users:     make(map[string]*User),
		CreatedAt: time.Now(),
		CreatedBy: creatorUsername,
		MaxUsers:  50, // จำกัด 50 คนต่อห้อง
		IsActive:  true,
	}

	rm.rooms[name] = room
	log.Printf("🏠 Room '%s' created by %s", name, creatorUsername)
	return room, nil
}

// JoinRoom adds a user to a room
func (rm *RoomManager) JoinRoom(user *User, roomName string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// ตรวจสอบว่าห้องมีอยู่หรือไม่
	room, exists := rm.rooms[roomName]
	if !exists {
		return fmt.Errorf("room '%s' does not exist", roomName)
	}

	// ตรวจสอบว่าห้องเต็มหรือไม่
	if len(room.Users) >= room.MaxUsers {
		return fmt.Errorf("room '%s' is full", roomName)
	}

	// ออกจากห้องเก่า (ถ้ามี)
	if user.CurrentRoom != "" {
		rm.leaveRoomInternal(user, user.CurrentRoom)
	}

	// เข้าห้องใหม่
	room.Users[user.Username] = user
	user.CurrentRoom = roomName

	log.Printf("🚪 User %s joined room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// LeaveRoom removes a user from a room
func (rm *RoomManager) LeaveRoom(user *User, roomName string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	return rm.leaveRoomInternal(user, roomName)
}

// leaveRoomInternal removes a user from a room (internal, assumes lock is held)
func (rm *RoomManager) leaveRoomInternal(user *User, roomName string) error {
	room, exists := rm.rooms[roomName]
	if !exists {
		return fmt.Errorf("room '%s' does not exist", roomName)
	}

	// ลบผู้ใช้จากห้อง
	delete(room.Users, user.Username)
	if user.CurrentRoom == roomName {
		user.CurrentRoom = ""
	}

	log.Printf("🚪 User %s left room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// GetRoom returns a room by name
func (rm *RoomManager) GetRoom(name string) (*Room, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	room, exists := rm.rooms[name]
	return room, exists
}

// GetRooms returns all active rooms
func (rm *RoomManager) GetRooms() []*Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	rooms := make([]*Room, 0, len(rm.rooms))
	for _, room := range rm.rooms {
		if room.IsActive {
			rooms = append(rooms, room)
		}
	}
	return rooms
}

// GetUsersInRoom returns all users in a specific room
func (rm *RoomManager) GetUsersInRoom(roomName string) []*User {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	room, exists := rm.rooms[roomName]
	if !exists {
		return []*User{}
	}
	
	users := make([]*User, 0, len(room.Users))
	for _, user := range room.Users {
		users = append(users, user)
	}
	return users
}

// GetRoomCount returns the number of active rooms
func (rm *RoomManager) GetRoomCount() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	count := 0
	for _, room := range rm.rooms {
		if room.IsActive {
			count++
		}
	}
	return count
}
type UserManager struct {
	users       map[string]*User  // connID -> User
	usersByName map[string]*User  // username -> User
	mutex       sync.RWMutex
}

// NewUserManager creates a new user manager
func NewUserManager() *UserManager {
	return &UserManager{
		users:       make(map[string]*User),
		usersByName: make(map[string]*User),
	}
}

// RegisterUser registers a new user with username validation
func (um *UserManager) RegisterUser(connID, username string) (*User, error) {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	// ตรวจสอบว่า username ว่างหรือไม่
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// ตรวจสอบว่า username ซ้ำหรือไม่
	if _, exists := um.usersByName[username]; exists {
		return nil, fmt.Errorf("username '%s' is already taken", username)
	}

	// สร้าง user ใหม่
	user := &User{
		ID:              generateUserID(),
		Username:        username,
		ConnID:          connID,
		JoinedAt:        time.Now(),
		LastActive:      time.Now(),
		IsAuthenticated: true,
	}

	// เก็บ user
	um.users[connID] = user
	um.usersByName[username] = user

	log.Printf("👤 User registered: %s (ConnID: %s)", username, connID)
	return user, nil
}

// UnregisterUser removes a user
func (um *UserManager) UnregisterUser(connID string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	user, exists := um.users[connID]
	if !exists {
		return fmt.Errorf("user not found for connection %s", connID)
	}

	// ลบจาก maps
	delete(um.users, connID)
	delete(um.usersByName, user.Username)

	log.Printf("👋 User unregistered: %s (ConnID: %s)", user.Username, connID)
	return nil
}

// GetUser returns a user by connection ID
func (um *UserManager) GetUser(connID string) (*User, bool) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	user, exists := um.users[connID]
	return user, exists
}

// GetUserByName returns a user by username
func (um *UserManager) GetUserByName(username string) (*User, bool) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	user, exists := um.usersByName[username]
	return user, exists
}

// IsUsernameAvailable checks if a username is available
func (um *UserManager) IsUsernameAvailable(username string) bool {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	_, exists := um.usersByName[username]
	return !exists
}

// GetAllUsers returns all registered users
func (um *UserManager) GetAllUsers() []*User {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	users := make([]*User, 0, len(um.users))
	for _, user := range um.users {
		users = append(users, user)
	}
	return users
}

// UpdateLastActive updates user's last active time
func (um *UserManager) UpdateLastActive(connID string) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	if user, exists := um.users[connID]; exists {
		user.LastActive = time.Now()
	}
}

// generateUserID creates a unique user ID
func generateUserID() string {
	return "user-" + time.Now().Format("20060102150405") + "-" + randomString(4)
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
		delete(cm.connections, conn.ID)
	}
}

// unregisterConnection removes a connection
func (cm *ConnectionManager) unregisterConnection(conn *Connection) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.connections[conn.ID]; exists {
		// ถ้ามี user ให้แจ้งเตือนคนอื่น
		if conn.User != nil && conn.User.IsAuthenticated {
			// ส่งข้อความแจ้งว่ามีคนออก
			leaveMsg := &Message{
				Type:      "user_left",
				Content:   fmt.Sprintf("👋 %s ออกจากระบบแล้ว", conn.User.Username),
				Sender:    "System",
				Username:  "System",
				Timestamp: time.Now(),
			}
			
			// Broadcast ข้อความแจ้งให้คนอื่นรู้
			cm.broadcastMessage(&BroadcastMessage{
				Message:   leaveMsg,
				ExcludeID: "", // ส่งให้ทุกคน
			})

			// ออกจากห้องปัจจุบัน
			if conn.User.CurrentRoom != "" {
				roomManager.LeaveRoom(conn.User, conn.User.CurrentRoom)
			}

			// ลบ user จาก user manager
			userManager.UnregisterUser(conn.ID)
		}

		delete(cm.connections, conn.ID)
		close(conn.Send)
		log.Printf("🗑️ Connection unregistered: %s (Total: %d)", conn.ID, len(cm.connections))
	}
}

// broadcastMessage sends a message to all connections except the sender
func (cm *ConnectionManager) broadcastMessage(broadcastMsg *BroadcastMessage) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

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

	for connID, conn := range cm.connections {
		// ไม่ส่งข้อความกลับไปยังผู้ส่ง
		if connID == excludeID {
			continue
		}

		// ตรวจสอบว่า connection มี user และอยู่ในห้องที่ถูกต้องหรือไม่
		if roomName != "" && conn.User != nil {
			if conn.User.CurrentRoom != roomName {
				continue // ไม่อยู่ในห้องเดียวกัน ข้าม
			}
		}

		select {
		case conn.Send <- []byte(formattedMessage):
			sentCount++
		default:
			// Connection ไม่ตอบสนอง ลบออก
			close(conn.Send)
			delete(cm.connections, connID)
			log.Printf("🔌 Removed unresponsive connection: %s", connID)
		}
	}

	if roomName != "" {
		log.Printf("📡 Broadcasted message to %d connections in room '%s' (excluded: %s)", sentCount, roomName, excludeID)
	} else {
		log.Printf("📡 Broadcasted message to %d connections (excluded: %s)", sentCount, excludeID)
	}
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
	cm.BroadcastToRoom(message, excludeID, "")
}

// BroadcastToRoom broadcasts a message to connections in a specific room
func (cm *ConnectionManager) BroadcastToRoom(message *Message, excludeID, roomName string) {
	broadcastMsg := &BroadcastMessage{
		Message:   message,
		ExcludeID: excludeID,
		RoomName:  roomName,
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

// Global connection manager, user manager, and room manager
var connectionManager *ConnectionManager
var userManager *UserManager
var roomManager *RoomManager

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

		// ดึง connection object
		connection, exists := connectionManager.GetConnection(connID)
		if !exists {
			log.Printf("❌ Connection not found: %s", connID)
			break
		}

		// ตรวจสอบว่า user authenticated หรือยัง
		if connection.User == nil || !connection.User.IsAuthenticated {
			// ยังไม่ authenticated - ใช้ข้อความเป็น username
			username := strings.TrimSpace(messageContent)
			
			// ตรวจสอบ username
			if username == "" {
				sendErrorMessage(connection, "❌ ชื่อผู้ใช้ไม่สามารถเว้นว่างได้ กรุณาระบุชื่อผู้ใช้:")
				continue
			}

			// ลองลงทะเบียน user
			user, err := userManager.RegisterUser(connID, username)
			if err != nil {
				sendErrorMessage(connection, fmt.Sprintf("❌ %s กรุณาเลือกชื่อผู้ใช้อื่น:", err.Error()))
				continue
			}

			// เก็บ user ใน connection
			connection.User = user

			// เข้าห้อง default อัตโนมัติ
			err = roomManager.JoinRoom(user, "general")
			if err != nil {
				log.Printf("❌ Failed to join default room: %v", err)
			}

			// ส่งข้อความต้อนรับ
			welcomeMsg := fmt.Sprintf("🎉 ยินดีต้อนรับ %s! คุณอยู่ในห้อง 'general' แล้ว", username)
			sendSystemMessage(connection, welcomeMsg)

			// แจ้งให้คนในห้องเดียวกันรู้ว่ามีคนเข้ามา
			joinMsg := &Message{
				Type:      "user_joined",
				Content:   fmt.Sprintf("👋 %s เข้าร่วมห้อง 'general' แล้ว", username),
				Sender:    "System",
				Username:  "System",
				Timestamp: time.Now(),
			}
			connectionManager.BroadcastToRoom(joinMsg, connID, "general")

		} else {
			// User authenticated แล้ว - ประมวลผลข้อความปกติ
			userManager.UpdateLastActive(connID)

			// สร้าง message object พร้อม username
			message := &Message{
				Type:      "text",
				Content:   messageContent,
				Sender:    clientAddr,
				Username:  connection.User.Username,
				Timestamp: time.Now(),
			}

			// Broadcast ข้อความไปยัง clients ในห้องเดียวกัน (ไม่รวมผู้ส่ง)
			connectionManager.BroadcastToRoom(message, connID, connection.User.CurrentRoom)
		}
	}
}

// sendSystemMessage sends a system message to a specific connection
func sendSystemMessage(conn *Connection, message string) {
	select {
	case conn.Send <- []byte(message):
	default:
		log.Printf("❌ Failed to send system message to %s", conn.ID)
	}
}

// sendErrorMessage sends an error message to a specific connection
func sendErrorMessage(conn *Connection, message string) {
	select {
	case conn.Send <- []byte(message):
	default:
		log.Printf("❌ Failed to send error message to %s", conn.ID)
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
	// สร้าง managers
	connectionManager = NewConnectionManager()
	userManager = NewUserManager()
	roomManager = NewRoomManager()

	// เริ่ม connection manager ใน goroutine
	go connectionManager.Run()

	// ตั้งค่า HTTP routes
	http.HandleFunc("/ws", handleWebSocket)

	// เสิร์ฟ static files สำหรับ test client
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	// เริ่มต้น server
	port := ":9090"
	log.Printf("� Startingt WebSocket Chat Server on port %s", port)
	log.Printf("📡 WebSocket endpoint: ws://localhost%s/ws", port)
	log.Printf("🌐 Test page: http://localhost%s", port)
	log.Printf("👥 Connection Manager: Ready")
	log.Printf("🔐 User Manager: Ready")
	log.Printf("🏠 Room Manager: Ready")

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
