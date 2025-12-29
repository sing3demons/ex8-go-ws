package chat

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"realtime-chat/internal/config"
	messagePkg "realtime-chat/internal/message"
	userPkg "realtime-chat/internal/user"
)

// commandService implements CommandService
type commandService struct {
	userService     UserService
	roomService     RoomService
	messageService  MessageService
	wsManager       WebSocketManager
	metrics         *config.ServerMetrics
	config          *config.ServerConfig
	configManager   *config.ConfigManager
	messageRepo     MessageRepository
	commands        map[string]*Command
}

// NewCommandService creates a new command service
func NewCommandService(
	userService UserService,
	roomService RoomService,
	messageService MessageService,
	wsManager WebSocketManager,
	metrics *config.ServerMetrics,
	cfg *config.ServerConfig,
	configManager *config.ConfigManager,
) CommandService {
	service := &commandService{
		userService:   userService,
		roomService:   roomService,
		messageService: messageService,
		wsManager:     wsManager,
		metrics:       metrics,
		config:        cfg,
		configManager: configManager,
		commands:      make(map[string]*Command),
	}

	// Register default commands
	service.registerDefaultCommands()

	return service
}

// SetMessageRepository sets the message repository
func (s *commandService) SetMessageRepository(repo MessageRepository) {
	s.messageRepo = repo
}

// RegisterCommand registers a new command
func (s *commandService) RegisterCommand(cmd *Command) {
	s.commands[cmd.Name] = cmd
}

// ExecuteCommand executes a command
func (s *commandService) ExecuteCommand(conn Connection, message string) error {
	// Parse command and arguments
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	commandName := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	// Find and execute command
	if cmd, exists := s.commands[commandName]; exists {
		return cmd.Handler(conn, args)
	}

	return fmt.Errorf("unknown command: /%s", commandName)
}

// GetCommands returns all registered commands
func (s *commandService) GetCommands() map[string]*Command {
	return s.commands
}

// registerDefaultCommands registers the default chat commands
func (s *commandService) registerDefaultCommands() {
	// Help command
	s.RegisterCommand(&Command{
		Name:        "help",
		Description: "Show available commands",
		Usage:       "/help",
		Handler:     s.handleHelp,
	})

	// Users command
	s.RegisterCommand(&Command{
		Name:        "users",
		Description: "List users in current room",
		Usage:       "/users",
		Handler:     s.handleUsers,
	})

	// Rooms command
	s.RegisterCommand(&Command{
		Name:        "rooms",
		Description: "List all available rooms",
		Usage:       "/rooms",
		Handler:     s.handleRooms,
	})

	// Join command
	s.RegisterCommand(&Command{
		Name:        "join",
		Description: "Join a room",
		Usage:       "/join <room_name>",
		Handler:     s.handleJoin,
	})

	// Leave command
	s.RegisterCommand(&Command{
		Name:        "leave",
		Description: "Leave current room",
		Usage:       "/leave",
		Handler:     s.handleLeave,
	})

	// Create command
	s.RegisterCommand(&Command{
		Name:        "create",
		Description: "Create a new room",
		Usage:       "/create <room_name>",
		Handler:     s.handleCreate,
	})

	// Stats command
	s.RegisterCommand(&Command{
		Name:        "stats",
		Description: "Show server statistics",
		Usage:       "/stats",
		Handler:     s.handleStats,
	})

	// History command (if MongoDB is enabled)
	if s.config.EnableMongoDB {
		s.RegisterCommand(&Command{
			Name:        "history",
			Description: "Get message history for current room",
			Usage:       "/history [limit]",
			Handler:     s.handleHistory,
		})

		s.RegisterCommand(&Command{
			Name:        "search",
			Description: "Search messages in current room",
			Usage:       "/search <query>",
			Handler:     s.handleSearch,
		})
	}
}

// Command handlers

func (s *commandService) handleHelp(conn Connection, args []string) error {
	var helpText strings.Builder
	helpText.WriteString("📋 Available Commands:\n")

	for _, cmd := range s.commands {
		helpText.WriteString(fmt.Sprintf("• %s - %s\n", cmd.Usage, cmd.Description))
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   helpText.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  "",
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleUsers(conn Connection, args []string) error {
	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	if chatUser.CurrentRoom == "" {
		return fmt.Errorf("you are not in any room")
	}

	users := s.roomService.GetUsersInRoom(chatUser.CurrentRoom)
	var userList strings.Builder
	userList.WriteString(fmt.Sprintf("👥 Users in room '%s' (%d users):\n", chatUser.CurrentRoom, len(users)))

	for _, u := range users {
		userList.WriteString(fmt.Sprintf("• %s\n", u.Username))
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   userList.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  chatUser.CurrentRoom,
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleRooms(conn Connection, args []string) error {
	rooms := s.roomService.GetRooms()
	var roomList strings.Builder
	roomList.WriteString(fmt.Sprintf("🏠 Available rooms (%d rooms):\n", len(rooms)))

	for _, room := range rooms {
		userCount := len(room.Users)
		roomList.WriteString(fmt.Sprintf("• %s (%d/%d users)\n", room.Name, userCount, room.MaxUsers))
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   roomList.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  "",
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleJoin(conn Connection, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("room name required. Usage: /join <room_name>")
	}

	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	roomName := args[0]

	// Leave current room if in one
	if chatUser.CurrentRoom != "" {
		if err := s.roomService.LeaveRoom(chatUser, chatUser.CurrentRoom); err != nil {
			log.Printf("⚠️ Failed to leave current room: %v", err)
		}
	}

	// Join new room
	if err := s.roomService.JoinRoom(chatUser, roomName); err != nil {
		return fmt.Errorf("failed to join room '%s': %v", roomName, err)
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   fmt.Sprintf("✅ Joined room '%s'", roomName),
		Sender:    "System",
		Username:  "System",
		RoomName:  roomName,
		Timestamp: time.Now(),
	}

	// Notify others in the room
	joinMsg := &messagePkg.Message{
		Type:      "user_joined",
		Content:   fmt.Sprintf("%s joined the room", chatUser.Username),
		Sender:    "System",
		Username:  "System",
		RoomName:  roomName,
		Timestamp: time.Now(),
	}

	s.messageService.BroadcastToRoom(joinMsg, conn.GetID(), roomName)

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		message.Content, message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleLeave(conn Connection, args []string) error {
	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	if chatUser.CurrentRoom == "" {
		return fmt.Errorf("you are not in any room")
	}

	roomName := chatUser.CurrentRoom

	// Leave room
	if err := s.roomService.LeaveRoom(chatUser, roomName); err != nil {
		return fmt.Errorf("failed to leave room: %v", err)
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   fmt.Sprintf("✅ Left room '%s'", roomName),
		Sender:    "System",
		Username:  "System",
		RoomName:  "",
		Timestamp: time.Now(),
	}

	// Notify others in the room
	leaveMsg := &messagePkg.Message{
		Type:      "user_left",
		Content:   fmt.Sprintf("%s left the room", chatUser.Username),
		Sender:    "System",
		Username:  "System",
		RoomName:  roomName,
		Timestamp: time.Now(),
	}

	s.messageService.BroadcastToRoom(leaveMsg, conn.GetID(), roomName)

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		message.Content, message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleCreate(conn Connection, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("room name required. Usage: /create <room_name>")
	}

	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	roomName := args[0]

	// Create room
	_, err := s.roomService.CreateRoom(roomName, chatUser.Username)
	if err != nil {
		return fmt.Errorf("failed to create room '%s': %v", roomName, err)
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   fmt.Sprintf("✅ Room '%s' created successfully", roomName),
		Sender:    "System",
		Username:  "System",
		RoomName:  "",
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		message.Content, message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleStats(conn Connection, args []string) error {
	rooms := s.roomService.GetRooms()
	users := s.userService.GetAllUsers()

	var stats strings.Builder
	stats.WriteString("📊 Server Statistics:\n")
	stats.WriteString(fmt.Sprintf("• Total Users: %d\n", len(users)))
	stats.WriteString(fmt.Sprintf("• Total Rooms: %d\n", len(rooms)))
	stats.WriteString(fmt.Sprintf("• Max Connections: %d\n", s.config.MaxConnections))
	stats.WriteString(fmt.Sprintf("• Max Rooms: %d\n", s.config.MaxRooms))

	if s.metrics != nil {
		stats.WriteString(fmt.Sprintf("• Messages Sent: %d\n", s.metrics.TotalMessages))
		stats.WriteString(fmt.Sprintf("• Commands Executed: %d\n", s.metrics.TotalCommands))
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   stats.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  "",
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleHistory(conn Connection, args []string) error {
	if s.messageRepo == nil {
		return fmt.Errorf("message history not available")
	}

	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	if chatUser.CurrentRoom == "" {
		return fmt.Errorf("you must be in a room to view history")
	}

	limit := 10 // default
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := s.messageRepo.GetMessageHistory(chatUser.CurrentRoom, limit)
	if err != nil {
		return fmt.Errorf("failed to get history: %v", err)
	}

	var history strings.Builder
	history.WriteString(fmt.Sprintf("📜 Last %d messages in '%s':\n", len(messages), chatUser.CurrentRoom))

	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		history.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.Content))
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   history.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  chatUser.CurrentRoom,
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}

func (s *commandService) handleSearch(conn Connection, args []string) error {
	if s.messageRepo == nil {
		return fmt.Errorf("message search not available")
	}

	if len(args) == 0 {
		return fmt.Errorf("search query required. Usage: /search <query>")
	}

	user := conn.GetUser()
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	chatUser, ok := user.(*userPkg.User)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	if chatUser.CurrentRoom == "" {
		return fmt.Errorf("you must be in a room to search messages")
	}

	query := strings.Join(args, " ")
	messages, err := s.messageRepo.SearchMessages(query, chatUser.CurrentRoom, 20)
	if err != nil {
		return fmt.Errorf("search failed: %v", err)
	}

	var results strings.Builder
	results.WriteString(fmt.Sprintf("🔍 Search results for '%s' in '%s' (%d results):\n", query, chatUser.CurrentRoom, len(messages)))

	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		results.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.Content))
	}

	if len(messages) == 0 {
		results.WriteString("No messages found.")
	}

	message := &messagePkg.Message{
		Type:      "system",
		Content:   results.String(),
		Sender:    "System",
		Username:  "System",
		RoomName:  chatUser.CurrentRoom,
		Timestamp: time.Now(),
	}

	return conn.SendMessage([]byte(fmt.Sprintf(`{"type":"system","content":"%s","sender":"System","timestamp":"%s"}`,
		strings.ReplaceAll(message.Content, "\n", "\\n"),
		message.Timestamp.Format(time.RFC3339))))
}