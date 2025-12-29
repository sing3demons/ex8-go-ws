package chat

import (
	messagePkg "realtime-chat/internal/message"
	"realtime-chat/internal/room"
	userPkg "realtime-chat/internal/user"
)

// UserService interface for user operations
type UserService interface {
	RegisterUser(connID, username string) (*userPkg.User, error)
	UnregisterUser(connID string) error
	GetUser(connID string) (*userPkg.User, bool)
	GetUserByName(username string) (*userPkg.User, bool)
	IsUsernameAvailable(username string) bool
	GetAllUsers() []*userPkg.User
	UpdateLastActive(connID string)
}

// RoomService interface for room operations
type RoomService interface {
	CreateRoom(name, creatorUsername string) (*room.Room, error)
	JoinRoom(user *userPkg.User, roomName string) error
	LeaveRoom(user *userPkg.User, roomName string) error
	GetRoom(name string) (*room.Room, bool)
	GetRooms() []*room.Room
	GetUsersInRoom(roomName string) []*userPkg.User
	GetRoomCount() int
}

// CommandService interface for command processing
type CommandService interface {
	RegisterCommand(cmd *Command)
	ExecuteCommand(conn Connection, message string) error
	GetCommands() map[string]*Command
	SetMessageRepository(repo MessageRepository)
}

// MessageService interface for message broadcasting
type MessageService interface {
	BroadcastMessage(message *messagePkg.Message, excludeID string)
	BroadcastToRoom(message *messagePkg.Message, excludeID, roomName string)
}

// MessageRepository interface for message persistence
type MessageRepository interface {
	SaveMessage(message *messagePkg.Message) error
	GetMessageHistory(roomName string, limit int) ([]*messagePkg.Message, error)
	GetRecentMessages(limit int) ([]*messagePkg.Message, error)
	GetUserMessageHistory(username string, limit int) ([]*messagePkg.Message, error)
	GetMessageCount(roomName string) (int64, error)
	SearchMessages(query string, roomName string, limit int) ([]*messagePkg.Message, error)
}

// WebSocketManager interface for WebSocket connection management
type WebSocketManager interface {
	AddConnection(conn interface{}) string
	RemoveConnection(connID string)
	GetConnection(connID string) (Connection, bool)
	BroadcastMessage(message interface{}, excludeID string)
	BroadcastToRoom(message interface{}, excludeID, roomName string)
	GetConnectionHealth(connID string) (interface{}, bool)
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
func (s *messageService) BroadcastMessage(message *messagePkg.Message, excludeID string) {
	s.wsManager.BroadcastMessage(message, excludeID)
}

// BroadcastToRoom broadcasts a message to connections in a specific room
func (s *messageService) BroadcastToRoom(message *messagePkg.Message, excludeID, roomName string) {
	s.wsManager.BroadcastToRoom(message, excludeID, roomName)
}