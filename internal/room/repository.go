package room

import (
	"fmt"
	"sync"
	"time"

	userPkg "realtime-chat/internal/user"
)

// Repository manages room data
type Repository interface {
	Create(name, creatorUsername string, maxUsers int) (*Room, error)
	GetByName(name string) (*Room, bool)
	GetAll() []*Room
	GetActiveRooms() []*Room
	GetUsersInRoom(roomName string) []*userPkg.User
	GetRoomCount() int
	JoinRoom(user *userPkg.User, roomName string) error
	LeaveRoom(user *userPkg.User, roomName string) error
}

// InMemoryRepository implements Repository using in-memory storage
type InMemoryRepository struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

// NewInMemoryRepository creates a new in-memory room repository
func NewInMemoryRepository() *InMemoryRepository {
	repo := &InMemoryRepository{
		rooms: make(map[string]*Room),
	}

	// สร้างห้อง default
	defaultRoom := &Room{
		Name:      "general",
		Users:     make(map[string]*userPkg.User),
		CreatedAt: time.Now(),
		CreatedBy: "System",
		MaxUsers:  100,
		IsActive:  true,
	}
	repo.rooms["general"] = defaultRoom

	return repo
}

// Create creates a new room
func (r *InMemoryRepository) Create(name, creatorUsername string, maxUsers int) (*Room, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.rooms[name]; exists {
		return nil, fmt.Errorf("room '%s' already exists", name)
	}

	room := &Room{
		Name:      name,
		Users:     make(map[string]*userPkg.User),
		CreatedAt: time.Now(),
		CreatedBy: creatorUsername,
		MaxUsers:  maxUsers,
		IsActive:  true,
	}

	r.rooms[name] = room
	return room, nil
}

// GetByName gets a room by name
func (r *InMemoryRepository) GetByName(name string) (*Room, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	room, exists := r.rooms[name]
	return room, exists
}

// GetAll returns all rooms
func (r *InMemoryRepository) GetAll() []*Room {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	rooms := make([]*Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// GetActiveRooms returns all active rooms
func (r *InMemoryRepository) GetActiveRooms() []*Room {
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
func (r *InMemoryRepository) GetUsersInRoom(roomName string) []*userPkg.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	room, exists := r.rooms[roomName]
	if !exists {
		return []*userPkg.User{}
	}

	users := make([]*userPkg.User, 0, len(room.Users))
	for _, user := range room.Users {
		users = append(users, user)
	}
	return users
}

// GetRoomCount returns the number of active rooms
func (r *InMemoryRepository) GetRoomCount() int {
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
func (r *InMemoryRepository) JoinRoom(user *userPkg.User, roomName string) error {
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
func (r *InMemoryRepository) LeaveRoom(user *userPkg.User, roomName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.leaveRoomInternal(user, roomName)
}

// leaveRoomInternal removes a user from a room (internal, assumes lock is held)
func (r *InMemoryRepository) leaveRoomInternal(user *userPkg.User, roomName string) error {
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