package chat

import (
	"fmt"
	"log"
	"strings"
	"time"

	"realtime-chat/internal/config"
	"realtime-chat/internal/security"
)

// UserService handles user business logic
type UserService interface {
	RegisterUser(connID, username string) (*User, error)
	UnregisterUser(connID string) error
	GetUser(connID string) (*User, bool)
	GetUserByName(username string) (*User, bool)
	IsUsernameAvailable(username string) bool
	GetAllUsers() []*User
	UpdateLastActive(connID string)
}

// RoomService handles room business logic
type RoomService interface {
	CreateRoom(name, creatorUsername string) (*Room, error)
	JoinRoom(user *User, roomName string) error
	LeaveRoom(user *User, roomName string) error
	GetRoom(name string) (*Room, bool)
	GetRooms() []*Room
	GetUsersInRoom(roomName string) []*User
	GetRoomCount() int
}

// CommandService handles command processing
type CommandService interface {
	RegisterCommand(cmd *Command)
	ExecuteCommand(conn Connection, message string) error
	GetCommands() map[string]*Command
}

// MessageService handles message broadcasting
type MessageService interface {
	BroadcastMessage(message *Message, excludeID string)
	BroadcastToRoom(message *Message, excludeID, roomName string)
}

// messageService implements MessageService
type messageService struct {
	wsManager WebSocketManager
}

// NewMessageService creates a new message service
func NewMessageService(wsManager WebSocketManager) MessageService {
	return &messageService{wsManager: wsManager}
}

// BroadcastMessage broadcasts a message to all connections except sender
func (s *messageService) BroadcastMessage(message *Message, excludeID string) {
	s.wsManager.BroadcastMessage(message, excludeID)
}

// BroadcastToRoom broadcasts a message to connections in a specific room
func (s *messageService) BroadcastToRoom(message *Message, excludeID, roomName string) {
	s.wsManager.BroadcastToRoom(message, excludeID, roomName)
}

// userService implements UserService
type userService struct {
	repo    UserRepository
	metrics *config.ServerMetrics
}

// NewUserService creates a new user service
func NewUserService(repo UserRepository, metrics *config.ServerMetrics) UserService {
	return &userService{
		repo:    repo,
		metrics: metrics,
	}
}

// RegisterUser registers a new user
func (s *userService) RegisterUser(connID, username string) (*User, error) {
	user, err := s.repo.Create(connID, username)
	if err != nil {
		return nil, err
	}

	log.Printf("👤 User registered: %s (ConnID: %s)", username, connID)
	s.metrics.IncrementUsers()
	return user, nil
}

// UnregisterUser removes a user
func (s *userService) UnregisterUser(connID string) error {
	err := s.repo.Delete(connID)
	if err != nil {
		return err
	}

	log.Printf("👋 User unregistered (ConnID: %s)", connID)
	return nil
}

// GetUser returns a user by connection ID
func (s *userService) GetUser(connID string) (*User, bool) {
	return s.repo.GetByID(connID)
}

// GetUserByName returns a user by username
func (s *userService) GetUserByName(username string) (*User, bool) {
	return s.repo.GetByUsername(username)
}

// IsUsernameAvailable checks if a username is available
func (s *userService) IsUsernameAvailable(username string) bool {
	return s.repo.IsUsernameAvailable(username)
}

// GetAllUsers returns all registered users
func (s *userService) GetAllUsers() []*User {
	return s.repo.GetAll()
}

// UpdateLastActive updates user's last active time
func (s *userService) UpdateLastActive(connID string) {
	s.repo.UpdateLastActive(connID)
}

// roomService implements RoomService
type roomService struct {
	repo      RoomRepository
	maxRooms  int
	maxUsers  int
	metrics   *config.ServerMetrics
}

// NewRoomService creates a new room service
func NewRoomService(repo RoomRepository, maxRooms, maxUsers int, metrics *config.ServerMetrics) RoomService {
	return &roomService{
		repo:     repo,
		maxRooms: maxRooms,
		maxUsers: maxUsers,
		metrics:  metrics,
	}
}

// CreateRoom creates a new room
func (s *roomService) CreateRoom(name, creatorUsername string) (*Room, error) {
	// ตรวจสอบ room limits
	if s.repo.GetRoomCount() >= s.maxRooms {
		return nil, fmt.Errorf("server room limit reached (%d/%d)", s.repo.GetRoomCount(), s.maxRooms)
	}

	room, err := s.repo.Create(name, creatorUsername, s.maxUsers)
	if err != nil {
		return nil, err
	}

	log.Printf("🏠 Room '%s' created by %s (%d/%d rooms)", name, creatorUsername, s.repo.GetRoomCount(), s.maxRooms)
	s.metrics.IncrementRooms()
	return room, nil
}

// JoinRoom adds a user to a room
func (s *roomService) JoinRoom(user *User, roomName string) error {
	err := s.repo.JoinRoom(user, roomName)
	if err != nil {
		return err
	}

	room, _ := s.repo.GetByName(roomName)
	log.Printf("🚪 User %s joined room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// LeaveRoom removes a user from a room
func (s *roomService) LeaveRoom(user *User, roomName string) error {
	err := s.repo.LeaveRoom(user, roomName)
	if err != nil {
		return err
	}

	room, _ := s.repo.GetByName(roomName)
	log.Printf("🚪 User %s left room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// GetRoom returns a room by name
func (s *roomService) GetRoom(name string) (*Room, bool) {
	return s.repo.GetByName(name)
}

// GetRooms returns all active rooms
func (s *roomService) GetRooms() []*Room {
	return s.repo.GetActiveRooms()
}

// GetUsersInRoom returns all users in a specific room
func (s *roomService) GetUsersInRoom(roomName string) []*User {
	return s.repo.GetUsersInRoom(roomName)
}

// GetRoomCount returns the number of active rooms
func (s *roomService) GetRoomCount() int {
	return s.repo.GetRoomCount()
}

// commandService implements CommandService
type commandService struct {
	commands    map[string]*Command
	userService UserService
	roomService RoomService
	msgService  MessageService
	wsManager   WebSocketManager
	metrics     *config.ServerMetrics
	config      *config.ServerConfig
	validator   *security.InputValidator
}

// NewCommandService creates a new command service
func NewCommandService(userService UserService, roomService RoomService, msgService MessageService, wsManager WebSocketManager, metrics *config.ServerMetrics, cfg *config.ServerConfig) CommandService {
	cs := &commandService{
		commands:    make(map[string]*Command),
		userService: userService,
		roomService: roomService,
		msgService:  msgService,
		wsManager:   wsManager,
		metrics:     metrics,
		config:      cfg,
		validator:   security.NewInputValidator(cfg),
	}

	// ลงทะเบียนคำสั่งพื้นฐาน
	cs.registerBuiltinCommands()

	return cs
}

// RegisterCommand registers a new command
func (s *commandService) RegisterCommand(cmd *Command) {
	s.commands[cmd.Name] = cmd
	log.Printf("📋 Command registered: /%s", cmd.Name)
}

// getUserFromConnection safely casts user from connection
func getUserFromConnection(conn Connection) (*User, bool) {
	user := conn.GetUser()
	if user == nil {
		return nil, false
	}
	if chatUser, ok := user.(*User); ok {
		return chatUser, true
	}
	return nil, false
}

// ExecuteCommand executes a command
func (s *commandService) ExecuteCommand(conn Connection, message string) error {
	if !strings.HasPrefix(message, "/") {
		return fmt.Errorf("not a command")
	}

	parts := strings.Fields(message)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	commandName := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	cmd, exists := s.commands[commandName]
	if !exists {
		return fmt.Errorf("unknown command: /%s", commandName)
	}

	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return fmt.Errorf("user not authenticated")
	}

	log.Printf("🎯 Executing command: /%s by %s", commandName, user.Username)
	s.metrics.IncrementCommands()
	return cmd.Handler(conn, args)
}

// GetCommands returns all available commands
func (s *commandService) GetCommands() map[string]*Command {
	commands := make(map[string]*Command)
	for name, cmd := range s.commands {
		commands[name] = cmd
	}
	return commands
}

// registerBuiltinCommands registers built-in commands
func (s *commandService) registerBuiltinCommands() {
	s.RegisterCommand(&Command{
		Name:        "help",
		Description: "แสดงรายการคำสั่งที่ใช้ได้",
		Usage:       "/help",
		Handler:     s.handleHelp,
	})

	s.RegisterCommand(&Command{
		Name:        "users",
		Description: "แสดงรายชื่อผู้ใช้ในห้องปัจจุบัน",
		Usage:       "/users",
		Handler:     s.handleUsers,
	})

	s.RegisterCommand(&Command{
		Name:        "rooms",
		Description: "แสดงรายการห้องทั้งหมด",
		Usage:       "/rooms",
		Handler:     s.handleRooms,
	})

	s.RegisterCommand(&Command{
		Name:        "join",
		Description: "เข้าร่วมห้องที่ระบุ",
		Usage:       "/join <room_name>",
		Handler:     s.handleJoin,
	})

	s.RegisterCommand(&Command{
		Name:        "leave",
		Description: "ออกจากห้องปัจจุบัน",
		Usage:       "/leave",
		Handler:     s.handleLeave,
	})

	s.RegisterCommand(&Command{
		Name:        "create",
		Description: "สร้างห้องใหม่",
		Usage:       "/create <room_name>",
		Handler:     s.handleCreate,
	})

	s.RegisterCommand(&Command{
		Name:        "stats",
		Description: "แสดงสถิติเซิร์ฟเวอร์",
		Usage:       "/stats",
		Handler:     s.handleStats,
	})

	s.RegisterCommand(&Command{
		Name:        "health",
		Description: "แสดงสถานะสุขภาพการเชื่อมต่อ",
		Usage:       "/health",
		Handler:     s.handleHealth,
	})

	s.RegisterCommand(&Command{
		Name:        "ratelimit",
		Description: "แสดงสถานะ rate limit ของคุณ",
		Usage:       "/ratelimit",
		Handler:     s.handleRateLimit,
	})
}

// Command handlers
func (s *commandService) handleHelp(conn Connection, args []string) error {
	commands := s.GetCommands()

	helpText := "📋 คำสั่งที่ใช้ได้:\n"
	helpText += "==================\n"

	for _, cmd := range commands {
		helpText += fmt.Sprintf("• %s - %s\n", cmd.Usage, cmd.Description)
	}

	return conn.SendMessage([]byte(helpText))
}

func (s *commandService) handleUsers(conn Connection, args []string) error {
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}
	
	if user.CurrentRoom == "" {
		return conn.SendMessage([]byte("❌ คุณไม่ได้อยู่ในห้องใดๆ"))
	}

	users := s.roomService.GetUsersInRoom(user.CurrentRoom)

	userText := fmt.Sprintf("👥 ผู้ใช้ในห้อง '%s' (%d คน):\n", user.CurrentRoom, len(users))
	userText += "========================\n"

	for _, u := range users {
		status := "🟢"
		if time.Since(u.LastActive) > 5*time.Minute {
			status = "🟡"
		}
		userText += fmt.Sprintf("• %s %s\n", status, u.Username)
	}

	return conn.SendMessage([]byte(userText))
}

func (s *commandService) handleRooms(conn Connection, args []string) error {
	rooms := s.roomService.GetRooms()
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}

	roomText := fmt.Sprintf("🏠 ห้องทั้งหมด (%d ห้อง):\n", len(rooms))
	roomText += "==================\n"

	for _, room := range rooms {
		userCount := len(room.Users)
		currentRoom := ""
		if user.CurrentRoom == room.Name {
			currentRoom = " (ปัจจุบัน)"
		}
		roomText += fmt.Sprintf("• %s - %d/%d คน%s\n", room.Name, userCount, room.MaxUsers, currentRoom)
	}

	return conn.SendMessage([]byte(roomText))
}

func (s *commandService) handleJoin(conn Connection, args []string) error {
	if len(args) == 0 {
		return conn.SendMessage([]byte("❌ กรุณาระบุชื่อห้อง: /join <room_name>"))
	}

	roomName := args[0]
	
	// Validate room name
	validatedRoomName, err := s.validator.ValidateRoomName(roomName)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ %s", err.Error())))
	}
	
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}

	if user.CurrentRoom == validatedRoomName {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ คุณอยู่ในห้อง '%s' อยู่แล้ว", validatedRoomName)))
	}

	_, exists := s.roomService.GetRoom(validatedRoomName)
	if !exists {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ ไม่พบห้อง '%s' ใช้ /create %s เพื่อสร้างห้องใหม่", validatedRoomName, validatedRoomName)))
	}

	// ออกจากห้องเก่า
	oldRoom := user.CurrentRoom
	if oldRoom != "" {
		leaveMsg := &Message{
			Type:      "user_left_room",
			Content:   fmt.Sprintf("👋 %s ออกจากห้อง '%s' แล้ว", user.Username, oldRoom),
			Sender:    "System",
			Username:  "System",
			Timestamp: time.Now(),
		}
		s.msgService.BroadcastToRoom(leaveMsg, conn.GetID(), oldRoom)
	}

	// เข้าห้องใหม่
	err = s.roomService.JoinRoom(user, validatedRoomName)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ ไม่สามารถเข้าห้อง '%s': %s", validatedRoomName, err.Error())))
	}

	// ส่งข้อความยืนยัน
	conn.SendMessage([]byte(fmt.Sprintf("✅ เข้าร่วมห้อง '%s' เรียบร้อยแล้ว", validatedRoomName)))

	// แจ้งคนในห้องใหม่
	joinMsg := &Message{
		Type:      "user_joined_room",
		Content:   fmt.Sprintf("👋 %s เข้าร่วมห้อง '%s' แล้ว", user.Username, validatedRoomName),
		Sender:    "System",
		Username:  "System",
		Timestamp: time.Now(),
	}
	s.msgService.BroadcastToRoom(joinMsg, conn.GetID(), validatedRoomName)

	return nil
}

func (s *commandService) handleLeave(conn Connection, args []string) error {
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}
	
	if user.CurrentRoom == "" {
		return conn.SendMessage([]byte("❌ คุณไม่ได้อยู่ในห้องใดๆ"))
	}

	if user.CurrentRoom == "general" {
		return conn.SendMessage([]byte("❌ ไม่สามารถออกจากห้อง 'general' ได้ ใช้ /join <room> เพื่อย้ายไปห้องอื่น"))
	}

	oldRoom := user.CurrentRoom

	// ออกจากห้องปัจจุบัน
	err := s.roomService.LeaveRoom(user, oldRoom)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ ไม่สามารถออกจากห้อง: %s", err.Error())))
	}

	// เข้าห้อง general อัตโนมัติ
	err = s.roomService.JoinRoom(user, "general")
	if err != nil {
		log.Printf("❌ Failed to auto-join general room: %v", err)
	}

	// ส่งข้อความยืนยัน
	conn.SendMessage([]byte(fmt.Sprintf("✅ ออกจากห้อง '%s' และกลับไปห้อง 'general' แล้ว", oldRoom)))

	// แจ้งคนในห้องเก่า
	leaveMsg := &Message{
		Type:      "user_left_room",
		Content:   fmt.Sprintf("👋 %s ออกจากห้อง '%s' แล้ว", user.Username, oldRoom),
		Sender:    "System",
		Username:  "System",
		Timestamp: time.Now(),
	}
	s.msgService.BroadcastToRoom(leaveMsg, conn.GetID(), oldRoom)

	// แจ้งคนในห้อง general
	joinMsg := &Message{
		Type:      "user_joined_room",
		Content:   fmt.Sprintf("👋 %s กลับมาห้อง 'general' แล้ว", user.Username),
		Sender:    "System",
		Username:  "System",
		Timestamp: time.Now(),
	}
	s.msgService.BroadcastToRoom(joinMsg, conn.GetID(), "general")

	return nil
}

func (s *commandService) handleCreate(conn Connection, args []string) error {
	if len(args) == 0 {
		return conn.SendMessage([]byte("❌ กรุณาระบุชื่อห้อง: /create <room_name>"))
	}

	roomName := args[0]
	
	// Validate room name
	validatedRoomName, err := s.validator.ValidateRoomName(roomName)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ %s", err.Error())))
	}
	
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}

	// สร้างห้องใหม่
	_, err = s.roomService.CreateRoom(validatedRoomName, user.Username)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ ไม่สามารถสร้างห้อง '%s': %s", validatedRoomName, err.Error())))
	}

	// เข้าห้องที่สร้างใหม่อัตโนมัติ
	oldRoom := user.CurrentRoom
	err = s.roomService.JoinRoom(user, validatedRoomName)
	if err != nil {
		return conn.SendMessage([]byte(fmt.Sprintf("❌ สร้างห้องสำเร็จแต่ไม่สามารถเข้าห้องได้: %s", err.Error())))
	}

	// ส่งข้อความยืนยัน
	conn.SendMessage([]byte(fmt.Sprintf("✅ สร้างห้อง '%s' และเข้าร่วมเรียบร้อยแล้ว", validatedRoomName)))

	// แจ้งคนในห้องเก่า
	if oldRoom != "" {
		leaveMsg := &Message{
			Type:      "user_left_room",
			Content:   fmt.Sprintf("👋 %s ออกจากห้อง '%s' เพื่อสร้างห้องใหม่", user.Username, oldRoom),
			Sender:    "System",
			Username:  "System",
			Timestamp: time.Now(),
		}
		s.msgService.BroadcastToRoom(leaveMsg, conn.GetID(), oldRoom)
	}

	// แจ้งทุกคนว่ามีห้องใหม่
	announceMsg := &Message{
		Type:      "room_created",
		Content:   fmt.Sprintf("🏠 %s สร้างห้อง '%s' ใหม่แล้ว ใช้ /join %s เพื่อเข้าร่วม", user.Username, validatedRoomName, validatedRoomName),
		Sender:    "System",
		Username:  "System",
		Timestamp: time.Now(),
	}
	s.msgService.BroadcastMessage(announceMsg, conn.GetID())

	return nil
}

// handleStats shows server statistics
func (s *commandService) handleStats(conn Connection, args []string) error {
	metrics := s.metrics.GetMetrics()
	uptime := time.Since(metrics.StartTime)
	
	statsText := "📊 สถิติเซิร์ฟเวอร์:\n"
	statsText += "==================\n"
	statsText += fmt.Sprintf("⏱️  เวลาทำงาน: %v\n", uptime.Round(time.Second))
	statsText += fmt.Sprintf("🔗 การเชื่อมต่อ: %d/%d (%d รวม)\n", metrics.ActiveConnections, s.config.MaxConnections, metrics.TotalConnections)
	statsText += fmt.Sprintf("👥 ผู้ใช้: %d คน\n", metrics.TotalUsers)
	statsText += fmt.Sprintf("🏠 ห้อง: %d/%d ห้อง\n", metrics.TotalRooms, s.config.MaxRooms)
	statsText += fmt.Sprintf("💬 ข้อความ: %d ข้อความ (%.2f/วินาที)\n", metrics.TotalMessages, metrics.MessageRate)
	statsText += fmt.Sprintf("📋 คำสั่ง: %d คำสั่ง\n", metrics.TotalCommands)
	statsText += fmt.Sprintf("📈 อัตราการเชื่อมต่อ: %.2f/วินาที\n", metrics.ConnectionRate)
	
	if !metrics.LastMessageTime.IsZero() {
		timeSinceLastMsg := time.Since(metrics.LastMessageTime)
		statsText += fmt.Sprintf("🕐 ข้อความล่าสุด: %v ที่แล้ว\n", timeSinceLastMsg.Round(time.Second))
	}
	
	return conn.SendMessage([]byte(statsText))
}

// handleHealth shows connection health information
func (s *commandService) handleHealth(conn Connection, args []string) error {
	// ดึงข้อมูล health ของ connection ปัจจุบัน
	if health, exists := s.wsManager.GetConnectionHealth(conn.GetID()); exists {
		uptime := time.Since(health.ConnectionStart)
		
		healthText := "💓 สถานะสุขภาพการเชื่อมต่อ:\n"
		healthText += "========================\n"
		healthText += fmt.Sprintf("🔗 Connection ID: %s\n", conn.GetID())
		healthText += fmt.Sprintf("⏱️  เวลาเชื่อมต่อ: %v\n", uptime.Round(time.Second))
		
		if health.IsHealthy {
			healthText += "💚 สถานะ: สุขภาพดี\n"
		} else {
			healthText += "💔 สถานะ: ไม่สุขภาพดี\n"
		}
		
		healthText += fmt.Sprintf("📤 Pings ส่ง: %d\n", health.PingsSent)
		healthText += fmt.Sprintf("📥 Pongs รับ: %d\n", health.PongsReceived)
		healthText += fmt.Sprintf("❌ Pongs พลาด: %d\n", health.MissedPongs)
		
		if !health.LastPingTime.IsZero() {
			healthText += fmt.Sprintf("📤 Ping ล่าสุด: %v ที่แล้ว\n", 
				time.Since(health.LastPingTime).Round(time.Second))
		}
		
		if !health.LastPongTime.IsZero() {
			healthText += fmt.Sprintf("📥 Pong ล่าสุด: %v ที่แล้ว\n", 
				time.Since(health.LastPongTime).Round(time.Second))
		}
		
		healthText += fmt.Sprintf("🔄 กิจกรรมล่าสุด: %v ที่แล้ว\n", 
			time.Since(health.LastActivity).Round(time.Second))
		
		return conn.SendMessage([]byte(healthText))
	}
	
	return conn.SendMessage([]byte("❌ ไม่สามารถดึงข้อมูลสุขภาพการเชื่อมต่อได้"))
}

// handleRateLimit shows rate limit status for the user
func (s *commandService) handleRateLimit(conn Connection, args []string) error {
	user, hasUser := getUserFromConnection(conn)
	if !hasUser {
		return conn.SendMessage([]byte("❌ ผู้ใช้ไม่ได้รับการยืนยันตัวตน"))
	}

	// Get rate limit status from handler (we need to access it somehow)
	// For now, we'll show the configuration
	rateLimitText := "⚡ สถานะ Rate Limit:\n"
	rateLimitText += "==================\n"
	rateLimitText += fmt.Sprintf("👤 ผู้ใช้: %s\n", user.Username)
	rateLimitText += fmt.Sprintf("📊 จำกัด: %d ข้อความต่อ %v\n", s.config.RateLimitMessages, s.config.RateLimitWindow)
	
	if s.config.EnableRateLimit {
		rateLimitText += "✅ Rate limiting: เปิดใช้งาน\n"
	} else {
		rateLimitText += "❌ Rate limiting: ปิดใช้งาน\n"
	}
	
	rateLimitText += fmt.Sprintf("📏 ความยาวข้อความสูงสุด: %d ตัวอักษร\n", s.config.MaxMessageLength)
	rateLimitText += fmt.Sprintf("👤 ความยาว username สูงสุด: %d ตัวอักษร\n", s.config.MaxUsernameLength)
	rateLimitText += fmt.Sprintf("🏠 ความยาวชื่อห้องสูงสุด: %d ตัวอักษร\n", s.config.MaxRoomNameLength)

	return conn.SendMessage([]byte(rateLimitText))
}