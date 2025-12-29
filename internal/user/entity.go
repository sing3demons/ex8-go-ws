package user

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