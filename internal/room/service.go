package room

import (
	"fmt"
	"log"

	"realtime-chat/internal/config"
	userPkg "realtime-chat/internal/user"
)

// Service handles room business logic
type Service interface {
	CreateRoom(name, creatorUsername string) (*Room, error)
	JoinRoom(user *userPkg.User, roomName string) error
	LeaveRoom(user *userPkg.User, roomName string) error
	GetRoom(name string) (*Room, bool)
	GetRooms() []*Room
	GetUsersInRoom(roomName string) []*userPkg.User
	GetRoomCount() int
}

// service implements Service
type service struct {
	repo      Repository
	maxRooms  int
	maxUsers  int
	metrics   *config.ServerMetrics
}

// NewService creates a new room service
func NewService(repo Repository, maxRooms, maxUsers int, metrics *config.ServerMetrics) Service {
	return &service{
		repo:     repo,
		maxRooms: maxRooms,
		maxUsers: maxUsers,
		metrics:  metrics,
	}
}

// CreateRoom creates a new room
func (s *service) CreateRoom(name, creatorUsername string) (*Room, error) {
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
func (s *service) JoinRoom(user *userPkg.User, roomName string) error {
	err := s.repo.JoinRoom(user, roomName)
	if err != nil {
		return err
	}

	room, _ := s.repo.GetByName(roomName)
	log.Printf("🚪 User %s joined room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// LeaveRoom removes a user from a room
func (s *service) LeaveRoom(user *userPkg.User, roomName string) error {
	err := s.repo.LeaveRoom(user, roomName)
	if err != nil {
		return err
	}

	room, _ := s.repo.GetByName(roomName)
	log.Printf("🚪 User %s left room '%s' (%d/%d users)", user.Username, roomName, len(room.Users), room.MaxUsers)
	return nil
}

// GetRoom returns a room by name
func (s *service) GetRoom(name string) (*Room, bool) {
	return s.repo.GetByName(name)
}

// GetRooms returns all active rooms
func (s *service) GetRooms() []*Room {
	return s.repo.GetActiveRooms()
}

// GetUsersInRoom returns all users in a specific room
func (s *service) GetUsersInRoom(roomName string) []*userPkg.User {
	return s.repo.GetUsersInRoom(roomName)
}

// GetRoomCount returns the number of active rooms
func (s *service) GetRoomCount() int {
	return s.repo.GetRoomCount()
}