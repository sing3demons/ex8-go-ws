package chat

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"realtime-chat/internal/config"
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
}

// WebSocketManager interface for WebSocket connection management
type WebSocketManager interface {
	AddConnection(conn *websocket.Conn) string
	RemoveConnection(connID string)
	GetConnection(connID string) (Connection, bool)
	BroadcastMessage(message interface{}, excludeID string)
	BroadcastToRoom(message interface{}, excludeID, roomName string)
	GetConnectionHealth(connID string) (*config.ConnectionHealth, bool)
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
	}
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

		// ตรวจสอบว่า user authenticated หรือยัง
		user := connection.GetUser()
		if user == nil {
			// ยังไม่ authenticated - ใช้ข้อความเป็น username
			username := strings.TrimSpace(messageContent)
			
			// ตรวจสอบ username
			if username == "" {
				h.sendErrorMessage(connection, "❌ ชื่อผู้ใช้ไม่สามารถเว้นว่างได้ กรุณาระบุชื่อผู้ใช้:")
				continue
			}

			// ลองลงทะเบียน user
			newUser, err := h.userService.RegisterUser(connID, username)
			if err != nil {
				h.sendErrorMessage(connection, fmt.Sprintf("❌ %s กรุณาเลือกชื่อผู้ใช้อื่น:", err.Error()))
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
			welcomeMsg := fmt.Sprintf("🎉 ยินดีต้อนรับ %s! คุณอยู่ในห้อง 'general' แล้ว", username)
			h.sendSystemMessage(connection, welcomeMsg)

			// แจ้งให้คนในห้องเดียวกันรู้ว่ามีคนเข้ามา
			joinMsg := &Message{
				Type:      "user_joined",
				Content:   fmt.Sprintf("👋 %s เข้าร่วมห้อง 'general' แล้ว", username),
				Sender:    "System",
				Username:  "System",
				Timestamp: time.Now(),
			}
			h.wsManager.BroadcastToRoom(joinMsg, connID, "general")

		} else {
			// User authenticated แล้ว - ประมวลผลข้อความปกติ
			if chatUser, ok := user.(*User); ok && chatUser.IsAuthenticated {
				h.userService.UpdateLastActive(connID)

				// ตรวจสอบว่าเป็นคำสั่งหรือไม่
				if strings.HasPrefix(messageContent, "/") {
					// ประมวลผลคำสั่ง
					err := h.commandService.ExecuteCommand(connection, messageContent)
					if err != nil {
						if err.Error() == "not a command" {
							// ไม่ใช่คำสั่ง ประมวลผลเป็นข้อความธรรมดา
						} else if strings.HasPrefix(err.Error(), "unknown command:") {
							h.sendErrorMessage(connection, fmt.Sprintf("❌ %s ใช้ /help เพื่อดูคำสั่งที่ใช้ได้", err.Error()))
							continue
						} else {
							h.sendErrorMessage(connection, fmt.Sprintf("❌ เกิดข้อผิดพลาด: %s", err.Error()))
							continue
						}
					} else {
						// คำสั่งทำงานสำเร็จ
						continue
					}
				}

				// ตรวจสอบว่าผู้ใช้อยู่ในห้องหรือไม่
				if chatUser.CurrentRoom == "" {
					h.sendErrorMessage(connection, "❌ คุณต้องอยู่ในห้องก่อนจึงจะส่งข้อความได้ ใช้ /join <room> เพื่อเข้าห้อง")
					continue
				}

				// สร้าง message object พร้อม username
				message := &Message{
					Type:      "text",
					Content:   messageContent,
					Sender:    clientAddr,
					Username:  chatUser.Username,
					Timestamp: time.Now(),
				}

				// Broadcast ข้อความไปยัง clients ในห้องเดียวกัน (ไม่รวมผู้ส่ง)
				h.wsManager.BroadcastToRoom(message, connID, chatUser.CurrentRoom)
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