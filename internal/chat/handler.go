package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"realtime-chat/internal/config"
	messagePkg "realtime-chat/internal/message"
	"realtime-chat/internal/security"
	userPkg "realtime-chat/internal/user"
	wsocket "realtime-chat/internal/websocket"
)

// Handler handles HTTP requests and WebSocket upgrades
type Handler struct {
	upgrader       websocket.Upgrader
	wsManager      WebSocketManager
	userService    UserService
	roomService    RoomService
	commandService CommandService
	messageService MessageService
	config         *config.ServerConfig
	rateLimiter    *config.RateLimiter
	validator      *security.InputValidator
	messageRepo    MessageRepository // Add message repository
}

// ClientMessage represents incoming messages from client
type ClientMessage struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Username string `json:"username,omitempty"`
	Room     string `json:"room,omitempty"`
	Command  string `json:"command,omitempty"`
	Query    string `json:"query,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// ServerMessage represents outgoing messages to client
type ServerMessage struct {
	Type      string                `json:"type"`
	Content   string                `json:"content,omitempty"`
	Sender    string                `json:"sender,omitempty"`
	Username  string                `json:"username,omitempty"`
	Room      string                `json:"room,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
	Users     []string              `json:"users,omitempty"`
	Rooms     []string              `json:"rooms,omitempty"`
	Messages  []*messagePkg.Message   `json:"messages,omitempty"`
	Message   string                `json:"message,omitempty"`
}

// NewHandler creates a new HTTP handler
func NewHandler(wsManager WebSocketManager, userService UserService, roomService RoomService, commandService CommandService, messageService MessageService, cfg *config.ServerConfig) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // อนุญาตให้ทุก origin เชื่อมต่อได้ (สำหรับการพัฒนา)
			},
		},
		wsManager:      wsManager,
		userService:    userService,
		roomService:    roomService,
		commandService: commandService,
		messageService: messageService,
		config:         cfg,
		rateLimiter:    config.NewRateLimiter(cfg),
		validator:      security.NewInputValidator(cfg),
		messageRepo:    nil, // Will be set later if MongoDB is enabled
	}
}

// SetMessageRepository sets the message repository for persistence
func (h *Handler) SetMessageRepository(repo MessageRepository) {
	h.messageRepo = repo
}

// HandleWebSocket handles WebSocket connection upgrades
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection เป็น WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// เพิ่ม connection ไปยัง manager
	connID := h.wsManager.AddConnection(conn)
	clientAddr := conn.RemoteAddr().String()
	log.Printf("🔗 New WebSocket connection: %s (ID: %s)", clientAddr, connID)

	// เริ่ม goroutines สำหรับ read และ write
	go h.handleRead(conn, connID, clientAddr)
	go h.handleWrite(conn, connID, clientAddr)
}

// handleRead จัดการการอ่านข้อความจาก client
func (h *Handler) handleRead(conn *websocket.Conn, connID, clientAddr string) {
	defer func() {
		h.wsManager.RemoveConnection(connID)
		conn.Close()
		log.Printf("🔌 Connection closed: %s (ID: %s)", clientAddr, connID)
	}()

	// ตั้งค่า read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		
		// อัพเดท health status เมื่อได้รับ pong
		if connection, exists := h.wsManager.GetConnection(connID); exists {
			if wsConn, ok := connection.(*wsocket.WebSocketConnection); ok {
				wsConn.Health.RecordPong()
			}
		}
		
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
		connection, exists := h.wsManager.GetConnection(connID)
		if !exists {
			log.Printf("❌ Connection not found: %s", connID)
			break
		}

		// Try to parse as JSON first
		var clientMsg ClientMessage
		var isJSON bool
		if err := json.Unmarshal(rawMessage, &clientMsg); err == nil && clientMsg.Type != "" {
			isJSON = true
		} else {
			// Fallback to plain text for backward compatibility
			clientMsg = ClientMessage{
				Type:    "message",
				Content: messageContent,
			}
		}

		// ตรวจสอบว่า user authenticated หรือยัง
		user := connection.GetUser()
		if user == nil {
			// Handle authentication
			var username string
			if isJSON && clientMsg.Type == "join" && clientMsg.Username != "" {
				username = clientMsg.Username
			} else {
				username = strings.TrimSpace(messageContent)
			}
			
			// Validate username
			validatedUsername, err := h.validator.ValidateUsername(username)
			if err != nil {
				h.sendJSONMessage(connection, ServerMessage{
					Type:      "error",
					Message:   err.Error(),
					Timestamp: time.Now(),
				})
				continue
			}

			// ลองลงทะเบียน user
			newUser, err := h.userService.RegisterUser(connID, validatedUsername)
			if err != nil {
				h.sendJSONMessage(connection, ServerMessage{
					Type:      "error",
					Message:   fmt.Sprintf("Username already taken: %s", err.Error()),
					Timestamp: time.Now(),
				})
				continue
			}

			// เก็บ user ใน connection
			connection.SetUser(newUser)

			// เข้าห้อง default อัตโนมัติ
			err = h.roomService.JoinRoom(newUser, "general")
			if err != nil {
				log.Printf("❌ Failed to join default room: %v", err)
			}

			// ส่งข้อความต้อนรับ
			h.sendJSONMessage(connection, ServerMessage{
				Type:      "system",
				Message:   fmt.Sprintf("Welcome %s! You joined room 'general'", validatedUsername),
				Timestamp: time.Now(),
			})

			// Send initial room and user lists
			h.sendRoomsList(connection)
			h.sendUsersList(connection, "general")

			// แจ้งให้คนในห้องเดียวกันรู้ว่ามีคนเข้ามา
			joinMsg := &messagePkg.Message{
				Type:      "user_joined",
				Content:   fmt.Sprintf("%s joined room 'general'", validatedUsername),
				Sender:    "System",
				Username:  "System",
				RoomName:  "general",
				Timestamp: time.Now(),
			}
			h.wsManager.BroadcastToRoom(joinMsg, connID, "general")

		} else {
			// User authenticated แล้ว - ประมวลผลข้อความ
			if chatUser, ok := user.(*userPkg.User); ok && chatUser.IsAuthenticated {
				h.userService.UpdateLastActive(connID)

				// Check rate limit
				if !h.rateLimiter.CheckRateLimit(chatUser.ID) {
					remaining, _, timeRemaining := h.rateLimiter.GetRateLimitStatus(chatUser.ID)
					h.sendJSONMessage(connection, ServerMessage{
						Type:      "error",
						Message:   fmt.Sprintf("Rate limit exceeded! You can send %d more messages in %v", remaining, timeRemaining.Round(time.Second)),
						Timestamp: time.Now(),
					})
					continue
				}

				// Handle different message types
				switch clientMsg.Type {
				case "message":
					h.handleChatMessage(connection, chatUser, clientMsg)
				case "command":
					h.handleCommand(connection, chatUser, clientMsg)
				case "join_room":
					h.handleJoinRoom(connection, chatUser, clientMsg)
				case "leave_room":
					h.handleLeaveRoom(connection, chatUser, clientMsg)
				case "create_room":
					h.handleCreateRoom(connection, chatUser, clientMsg)
				case "get_history":
					h.handleGetHistory(connection, chatUser, clientMsg)
				case "get_my_history":
					h.handleGetMyHistory(connection, chatUser, clientMsg)
				case "search_messages":
					h.handleSearchMessages(connection, chatUser, clientMsg)
				default:
					// Fallback to plain text message handling
					if clientMsg.Content != "" {
						if strings.HasPrefix(clientMsg.Content, "/") {
							// Handle as command
							clientMsg.Type = "command"
							clientMsg.Command = clientMsg.Content
							h.handleCommand(connection, chatUser, clientMsg)
						} else {
							// Handle as regular message
							h.handleChatMessage(connection, chatUser, clientMsg)
						}
					}
				}
			}
		}
	}
}

// handleWrite จัดการการเขียนข้อความไปยัง client
func (h *Handler) handleWrite(conn *websocket.Conn, connID, clientAddr string) {
	// ใช้ heartbeat interval จาก config
	ticker := time.NewTicker(h.config.HeartbeatInterval)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	// รอให้ connection ถูก register ก่อน
	time.Sleep(100 * time.Millisecond)

	// ดึง connection object จาก manager
	connection, exists := h.wsManager.GetConnection(connID)
	if !exists {
		log.Printf("❌ Connection not found: %s", connID)
		return
	}

	// Get the send channel through type assertion
	type SendChannelProvider interface {
		GetSendChannel() chan []byte
	}

	sendProvider, ok := connection.(SendChannelProvider)
	if !ok {
		log.Printf("❌ Connection does not provide send channel: %s", connID)
		return
	}

	sendChan := sendProvider.GetSendChannel()

	for {
		select {
		case message, ok := <-sendChan:
			if !ok {
				// Channel ถูกปิด - connection หมดอายุ
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			// ส่งข้อความไปยัง client
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Failed to send message to %s: %v", clientAddr, err)
				return
			}

		case <-ticker.C:
			// ตรวจสอบว่า connection ยังอยู่ใน manager หรือไม่
			if _, exists := h.wsManager.GetConnection(connID); !exists {
				// Connection ถูกลบจาก manager แล้ว - หยุดส่ง ping
				return
			}

			// ส่ง ping เพื่อ keep connection alive
			conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			
			// บันทึก ping ใน health tracker
			if wsConn, ok := connection.(*wsocket.WebSocketConnection); ok {
				wsConn.Health.RecordPing()
			}
			
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Failed to send ping to %s: %v", clientAddr, err)
				return
			}
			
			log.Printf("💓 Sent heartbeat ping to %s", clientAddr)
		}
	}
}

// sendSystemMessage sends a system message to a specific connection
func (h *Handler) sendSystemMessage(conn Connection, message string) {
	err := conn.SendMessage([]byte(message))
	if err != nil {
		log.Printf("❌ Failed to send system message to %s", conn.GetID())
	}
}

// sendErrorMessage sends an error message to a specific connection
func (h *Handler) sendErrorMessage(conn Connection, message string) {
	err := conn.SendMessage([]byte(message))
	if err != nil {
		log.Printf("❌ Failed to send error message to %s", conn.GetID())
	}
}

// sendJSONMessage sends a JSON message to a specific connection
func (h *Handler) sendJSONMessage(conn Connection, message ServerMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal JSON message: %v", err)
		return
	}
	
	err = conn.SendMessage(data)
	if err != nil {
		log.Printf("❌ Failed to send JSON message to %s: %v", conn.GetID(), err)
	}
}

// broadcastJSONToRoom broadcasts a JSON message to all connections in a room
func (h *Handler) broadcastJSONToRoom(message ServerMessage, excludeID, roomName string) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal JSON broadcast message: %v", err)
		return
	}
	
	h.wsManager.BroadcastToRoom(data, excludeID, roomName)
}

// handleChatMessage handles regular chat messages
func (h *Handler) handleChatMessage(conn Connection, user *userPkg.User, msg ClientMessage) {
	// ตรวจสอบว่าผู้ใช้อยู่ในห้องหรือไม่
	if user.CurrentRoom == "" {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "You must be in a room to send messages. Use /join <room> to join a room",
			Timestamp: time.Now(),
		})
		return
	}

	// Validate message content
	validatedMessage, err := h.validator.ValidateMessage(msg.Content)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// สร้าง message object
	message := &messagePkg.Message{
		Type:      "message",
		Content:   validatedMessage,
		Sender:    conn.GetID(),
		Username:  user.Username,
		RoomName:  user.CurrentRoom,
		Timestamp: time.Now(),
	}

	// Save message to database if MongoDB is enabled
	if h.messageRepo != nil {
		if err := h.messageRepo.SaveMessage(message); err != nil {
			log.Printf("⚠️ Failed to save message to database: %v", err)
		}
	}

	// Create server message for broadcast
	serverMsg := &messagePkg.Message{
		Type:      "message",
		Content:   validatedMessage,
		Sender:    conn.GetID(),
		Username:  user.Username,
		RoomName:  user.CurrentRoom,
		Timestamp: time.Now(),
	}

	// Broadcast to room (excluding sender)
	h.wsManager.BroadcastToRoom(serverMsg, conn.GetID(), user.CurrentRoom)
}

// handleCommand handles command messages
func (h *Handler) handleCommand(conn Connection, user *userPkg.User, msg ClientMessage) {
	var command string
	if msg.Command != "" {
		command = msg.Command
	} else {
		command = msg.Content
	}

	// Validate command
	validatedCommand, err := h.validator.ValidateCommand(command)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Invalid command: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	// Execute command
	err = h.commandService.ExecuteCommand(conn, validatedCommand)
	if err != nil {
		if err.Error() == "not a command" {
			// Handle as regular message
			msg.Type = "message"
			h.handleChatMessage(conn, user, msg)
		} else if strings.HasPrefix(err.Error(), "unknown command:") {
			h.sendJSONMessage(conn, ServerMessage{
				Type:      "error",
				Message:   fmt.Sprintf("%s Use /help to see available commands", err.Error()),
				Timestamp: time.Now(),
			})
		} else {
			h.sendJSONMessage(conn, ServerMessage{
				Type:      "error",
				Message:   fmt.Sprintf("Command error: %s", err.Error()),
				Timestamp: time.Now(),
			})
		}
	}
}

// handleJoinRoom handles room joining
func (h *Handler) handleJoinRoom(conn Connection, user *userPkg.User, msg ClientMessage) {
	if msg.Room == "" {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Room name is required",
			Timestamp: time.Now(),
		})
		return
	}

	// Leave current room if in one
	if user.CurrentRoom != "" {
		h.roomService.LeaveRoom(user, user.CurrentRoom)
	}

	// Join new room
	err := h.roomService.JoinRoom(user, msg.Room)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Failed to join room: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	// Send confirmation
	h.sendJSONMessage(conn, ServerMessage{
		Type:      "room_joined",
		Room:      msg.Room,
		Timestamp: time.Now(),
	})

	// Update room and user lists
	h.sendRoomsList(conn)
	h.sendUsersList(conn, msg.Room)
}

// handleLeaveRoom handles room leaving
func (h *Handler) handleLeaveRoom(conn Connection, user *userPkg.User, msg ClientMessage) {
	if user.CurrentRoom == "" {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "You are not in any room",
			Timestamp: time.Now(),
		})
		return
	}

	roomName := user.CurrentRoom
	err := h.roomService.LeaveRoom(user, roomName)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Failed to leave room: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	h.sendJSONMessage(conn, ServerMessage{
		Type:      "room_left",
		Room:      roomName,
		Timestamp: time.Now(),
	})
}

// handleCreateRoom handles room creation
func (h *Handler) handleCreateRoom(conn Connection, user *userPkg.User, msg ClientMessage) {
	if msg.Room == "" {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Room name is required",
			Timestamp: time.Now(),
		})
		return
	}

	// Create room (this would need to be implemented in room service)
	// For now, just send success message
	h.sendJSONMessage(conn, ServerMessage{
		Type:      "room_created",
		Room:      msg.Room,
		Timestamp: time.Now(),
	})

	// Update room list
	h.sendRoomsList(conn)
}

// handleGetHistory handles message history requests
func (h *Handler) handleGetHistory(conn Connection, user *userPkg.User, msg ClientMessage) {
	if h.messageRepo == nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Message history not available",
			Timestamp: time.Now(),
		})
		return
	}

	limit := msg.Limit
	if limit <= 0 {
		limit = 25
	}

	roomName := msg.Room
	if roomName == "" {
		roomName = user.CurrentRoom
	}

	messages, err := h.messageRepo.GetMessageHistory(roomName, limit)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Failed to get history: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	h.sendJSONMessage(conn, ServerMessage{
		Type:      "history",
		Messages:  messages,
		Timestamp: time.Now(),
	})
}

// handleGetMyHistory handles user's message history requests
func (h *Handler) handleGetMyHistory(conn Connection, user *userPkg.User, msg ClientMessage) {
	if h.messageRepo == nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Message history not available",
			Timestamp: time.Now(),
		})
		return
	}

	limit := msg.Limit
	if limit <= 0 {
		limit = 25
	}

	messages, err := h.messageRepo.GetUserMessageHistory(user.Username, limit)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Failed to get your history: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	h.sendJSONMessage(conn, ServerMessage{
		Type:      "history",
		Messages:  messages,
		Timestamp: time.Now(),
	})
}

// handleSearchMessages handles message search requests
func (h *Handler) handleSearchMessages(conn Connection, user *userPkg.User, msg ClientMessage) {
	if h.messageRepo == nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Message search not available",
			Timestamp: time.Now(),
		})
		return
	}

	if msg.Query == "" {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   "Search query is required",
			Timestamp: time.Now(),
		})
		return
	}

	roomName := msg.Room
	if roomName == "" {
		roomName = user.CurrentRoom
	}

	messages, err := h.messageRepo.SearchMessages(msg.Query, roomName, 50)
	if err != nil {
		h.sendJSONMessage(conn, ServerMessage{
			Type:      "error",
			Message:   fmt.Sprintf("Search failed: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return
	}

	h.sendJSONMessage(conn, ServerMessage{
		Type:      "search_results",
		Messages:  messages,
		Timestamp: time.Now(),
	})
}

// sendRoomsList sends the list of available rooms
func (h *Handler) sendRoomsList(conn Connection) {
	// This would need to be implemented to get actual room list
	// For now, send a basic list
	rooms := []string{"general"} // This should come from room service
	
	h.sendJSONMessage(conn, ServerMessage{
		Type:      "rooms_list",
		Rooms:     rooms,
		Timestamp: time.Now(),
	})
}

// sendUsersList sends the list of users in a room
func (h *Handler) sendUsersList(conn Connection, roomName string) {
	// This would need to be implemented to get actual user list
	// For now, send empty list
	users := []string{} // This should come from room service
	
	h.sendJSONMessage(conn, ServerMessage{
		Type:      "users_list",
		Users:     users,
		Timestamp: time.Now(),
	})
}