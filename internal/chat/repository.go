package chat

import (
	"fmt"
	"sync"
	"time"
)

// UserRepository manages user data
type UserRepository interface {
	Create(connID, username string) (*User, error)
	GetByID(connID string) (*User, bool)
	GetByUsername(username string) (*User, bool)
	Delete(connID string) error
	IsUsernameAvailable(username string) bool
	GetAll() []*User
	UpdateLastActive(connID string)
}

// RoomRepository manages room data
type RoomRepository interface {
	Create(name, creatorUsername string, maxUsers int) (*Room, error)
	GetByName(name string) (*Room, bool)
	GetAll() []*Room
	GetActiveRooms() []*Room
	GetUsersInRoom(roomName string) []*User
	GetRoomCount() int
	JoinRoom(user *User, roomName string) error
	LeaveRoom(user *User, roomName string) error
}

// InMemoryUserRepository implements UserRepository using in-memory storage
type InMemoryUserRepository struct {
	users       map[string]*User // connID -> User
	usersByName map[string]*User // username -> User
	mutex       sync.RWMutex
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:       make(map[string]*User),
		usersByName: make(map[string]*User),
	}
}

// Create creates a new user
func (r *InMemoryUserRepository) Create(connID, username string) (*User, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if _, exists := r.usersByName[username]; exists {
		return nil, fmt.Errorf("username '%s' is already taken", username)
	}

	user := &User{
		ID:              generateUserID(),
		Username:        username,
		ConnID:          connID,
		JoinedAt:        time.Now(),
		LastActive:      time.Now(),
		IsAuthenticated: true,
	}

	r.users[connID] = user
	r.usersByName[username] = user

	return user, nil
}

// GetByID gets a user by connection ID
func (r *InMemoryUserRepository) GetByID(connID string) (*User, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	user, exists := r.users[connID]
	return user, exists
}

// GetByUsername gets a user by username
func (r *InMemoryUserRepository) GetByUsername(username string) (*User, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	user, exists := r.usersByName[username]
	return user, exists
}

// Delete removes a user
func (r *InMemoryUserRepository) Delete(connID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[connID]
	if !exists {
		return fmt.Errorf("user not found for connection %s", connID)
	}

	delete(r.users, connID)
	delete(r.usersByName, user.Username)

	return nil
}

// IsUsernameAvailable checks if a username is available
func (r *InMemoryUserRepository) IsUsernameAvailable(username string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, exists := r.usersByName[username]
	return !exists
}

// GetAll returns all users
func (r *InMemoryUserRepository) GetAll() []*User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users
}

// UpdateLastActive updates user's last active time
func (r *InMemoryUserRepository) UpdateLastActive(connID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if user, exists := r.users[connID]; exists {
		user.LastActive = time.Now()
	}
}

// InMemoryRoomRepository implements RoomRepository using in-memory storage
type InMemoryRoomRepository struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// NewInMemoryRoomRepository creates a new in-memory room repository
func NewInMemoryRoomRepository() *InMemoryRoomRepository {
	repo := &InMemoryRoomRepository{
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
	repo.rooms["general"] = defaultRoom

	return repo
}

// Create creates a new room
func (r *InMemoryRoomRepository) Create(name, creatorUsername string, maxUsers int) (*Room, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.rooms[name]; exists {
		return nil, fmt.Errorf("room '%s' already exists", name)
	}

	room := &Room{
		Name:      name,
		Users:     make(map[string]*User),
		CreatedAt: time.Now(),
		CreatedBy: creatorUsername,
		MaxUsers:  maxUsers,
		IsActive:  true,
	}

	r.rooms[name] = room
	return room, nil
}

// GetByName gets a room by name
func (r *InMemoryRoomRepository) GetByName(name string) (*Room, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	room, exists := r.rooms[name]
	return room, exists
}

// GetAll returns all rooms
func (r *InMemoryRoomRepository) GetAll() []*Room {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rooms := make([]*Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// GetActiveRooms returns all active rooms
func (r *InMemoryRoomRepository) GetActiveRooms() []*Room {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rooms := make([]*Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		if room.IsActive {
			rooms = append(rooms, room)
		}
	}
	return rooms
}

// GetUsersInRoom returns all users in a specific room
func (r *InMemoryRoomRepository) GetUsersInRoom(roomName string) []*User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	room, exists := r.rooms[roomName]
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
func (r *InMemoryRoomRepository) GetRoomCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, room := range r.rooms {
		if room.IsActive {
			count++
		}
	}
	return count
}

// JoinRoom adds a user to a room
func (r *InMemoryRoomRepository) JoinRoom(user *User, roomName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	room, exists := r.rooms[roomName]
	if !exists {
		return fmt.Errorf("room '%s' does not exist", roomName)
	}

	if len(room.Users) >= room.MaxUsers {
		return fmt.Errorf("room '%s' is full", roomName)
	}

	// ออกจากห้องเก่า (ถ้ามี)
	if user.CurrentRoom != "" {
		r.leaveRoomInternal(user, user.CurrentRoom)
	}

	// เข้าห้องใหม่
	room.Users[user.Username] = user
	user.CurrentRoom = roomName

	return nil
}

// LeaveRoom removes a user from a room
func (r *InMemoryRoomRepository) LeaveRoom(user *User, roomName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.leaveRoomInternal(user, roomName)
}

// leaveRoomInternal removes a user from a room (internal, assumes lock is held)
func (r *InMemoryRoomRepository) leaveRoomInternal(user *User, roomName string) error {
	room, exists := r.rooms[roomName]
	if !exists {
		return fmt.Errorf("room '%s' does not exist", roomName)
	}

	delete(room.Users, user.Username)
	if user.CurrentRoom == roomName {
		user.CurrentRoom = ""
	}

	return nil
}

// generateUserID creates a unique user ID
func generateUserID() string {
	return "user-" + time.Now().Format("20060102150405") + "-" + randomString(4)
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