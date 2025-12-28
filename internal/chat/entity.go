package chat

import (
	"time"
)

// User represents a chat user
type User struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	ConnID          string    `json:"conn_id"`
	CurrentRoom     string    `json:"current_room"`
	JoinedAt        time.Time `json:"joined_at"`
	LastActive      time.Time `json:"last_active"`
	IsAuthenticated bool      `json:"is_authenticated"`
}

// GetIsAuthenticated returns the authentication status
func (u *User) GetIsAuthenticated() bool {
	return u.IsAuthenticated
}

// GetUsername returns the username
func (u *User) GetUsername() string {
	return u.Username
}

// GetCurrentRoom returns the current room
func (u *User) GetCurrentRoom() string {
	return u.CurrentRoom
}

// Room represents a chat room
type Room struct {
	Name      string            `json:"name"`
	Users     map[string]*User  `json:"users"`
	CreatedAt time.Time         `json:"created_at"`
	CreatedBy string            `json:"created_by"`
	MaxUsers  int               `json:"max_users"`
	IsActive  bool              `json:"is_active"`
}

// Message represents a message to be broadcasted
type Message struct {
	ID        string    `json:"id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Username  string    `json:"username"`
	RoomName  string    `json:"room_name"`
	Timestamp time.Time `json:"timestamp"`
}

// GetType returns the message type
func (m *Message) GetType() string {
	return m.Type
}

// GetContent returns the message content
func (m *Message) GetContent() string {
	return m.Content
}

// GetSender returns the message sender
func (m *Message) GetSender() string {
	return m.Sender
}

// GetUsername returns the message username
func (m *Message) GetUsername() string {
	return m.Username
}

// GetTimestamp returns the message timestamp
func (m *Message) GetTimestamp() time.Time {
	return m.Timestamp
}

// BroadcastMessage represents a message with exclusion info
type BroadcastMessage struct {
	Message   *Message
	ExcludeID string // ID ของ connection ที่ไม่ต้องการส่งไป
	RoomName  string // ชื่อห้องที่จะส่งข้อความ (ถ้าว่างจะส่งให้ทุกคน)
}

// Command represents a chat command
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(conn Connection, args []string) error
}

// Connection interface for WebSocket connections
type Connection interface {
	GetID() string
	GetUser() interface{}
	SetUser(user interface{})
	SendMessage(message []byte) error
	Close() error
}