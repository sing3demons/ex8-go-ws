package user

import (
	"log"

	"realtime-chat/internal/config"
)

// Service handles user business logic
type Service interface {
	RegisterUser(connID, username string) (*User, error)
	UnregisterUser(connID string) error
	GetUser(connID string) (*User, bool)
	GetUserByName(username string) (*User, bool)
	IsUsernameAvailable(username string) bool
	GetAllUsers() []*User
	UpdateLastActive(connID string)
}

// service implements Service
type service struct {
	repo    Repository
	metrics *config.ServerMetrics
}

// NewService creates a new user service
func NewService(repo Repository, metrics *config.ServerMetrics) Service {
	return &service{
		repo:    repo,
		metrics: metrics,
	}
}

// RegisterUser registers a new user
func (s *service) RegisterUser(connID, username string) (*User, error) {
	user, err := s.repo.Create(connID, username)
	if err != nil {
		return nil, err
	}

	log.Printf("👤 User registered: %s (ConnID: %s)", username, connID)
	s.metrics.IncrementUsers()
	return user, nil
}

// UnregisterUser removes a user
func (s *service) UnregisterUser(connID string) error {
	err := s.repo.Delete(connID)
	if err != nil {
		return err
	}

	log.Printf("👋 User unregistered (ConnID: %s)", connID)
	return nil
}

// GetUser returns a user by connection ID
func (s *service) GetUser(connID string) (*User, bool) {
	return s.repo.GetByID(connID)
}

// GetUserByName returns a user by username
func (s *service) GetUserByName(username string) (*User, bool) {
	return s.repo.GetByUsername(username)
}

// IsUsernameAvailable checks if a username is available
func (s *service) IsUsernameAvailable(username string) bool {
	return s.repo.IsUsernameAvailable(username)
}

// GetAllUsers returns all registered users
func (s *service) GetAllUsers() []*User {
	return s.repo.GetAll()
}

// UpdateLastActive updates user's last active time
func (s *service) UpdateLastActive(connID string) {
	s.repo.UpdateLastActive(connID)
}