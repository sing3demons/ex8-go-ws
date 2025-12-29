package chat

// BroadcastMessage represents a message with exclusion info
type BroadcastMessage struct {
	Message   interface{}
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