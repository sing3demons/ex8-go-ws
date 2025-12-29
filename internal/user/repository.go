package user

import (
	"fmt"
	"sync"
	"time"
)

// Repository manages user data
type Repository interface {
	Create(connID, username string) (*User, error)
	GetByID(connID string) (*User, bool)
	GetByUsername(username string) (*User, bool)
	Delete(connID string) error
	IsUsernameAvailable(username string) bool
	GetAll() []*User
	UpdateLastActive(connID string)
}

// InMemoryRepository implements Repository using in-memory storage
type InMemoryRepository struct {
	users       map[string]*User // connID -> User
	usersByName map[string]*User // username -> User
	mutex       sync.RWMutex
}

// NewInMemoryRepository creates a new in-memory user repository
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users:       make(map[string]*User),
		usersByName: make(map[string]*User),
	}
}

// Create creates a new user
func (r *InMemoryRepository) Create(connID, username string) (*User, error) {
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
func (r *InMemoryRepository) GetByID(connID string) (*User, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	user, exists := r.users[connID]
	return user, exists
}

// GetByUsername gets a user by username
func (r *InMemoryRepository) GetByUsername(username string) (*User, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	user, exists := r.usersByName[username]
	return user, exists
}

// Delete removes a user
func (r *InMemoryRepository) Delete(connID string) error {
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
func (r *InMemoryRepository) IsUsernameAvailable(username string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, exists := r.usersByName[username]
	return !exists
}

// GetAll returns all users
func (r *InMemoryRepository) GetAll() []*User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users
}

// UpdateLastActive updates user's last active time
func (r *InMemoryRepository) UpdateLastActive(connID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if user, exists := r.users[connID]; exists {
		user.LastActive = time.Now()
	}
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