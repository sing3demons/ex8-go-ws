package room

import (
	"time"

	userPkg "realtime-chat/internal/user"
)

// Room represents a chat room
type Room struct {
	Name      string                     `json:"name"`
	Users     map[string]*userPkg.User   `json:"users"`
	CreatedAt time.Time                  `json:"created_at"`
	CreatedBy string                     `json:"created_by"`
	MaxUsers  int                        `json:"max_users"`
	IsActive  bool                       `json:"is_active"`
}