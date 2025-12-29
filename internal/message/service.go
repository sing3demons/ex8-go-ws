package message

import (
	"fmt"
	"log"
	"time"
)

// Service handles message business logic
type Service interface {
	// Basic message operations
	SendMessage(message *Message) error
	GetMessage(messageID string) (*Message, error)
	UpdateMessage(message *Message) error
	DeleteMessage(messageID string) error
	
	// Message history and retrieval
	GetMessageHistory(roomName string, limit int) ([]*Message, error)
	GetRecentMessages(limit int) ([]*Message, error)
	GetUserMessageHistory(username string, limit int) ([]*Message, error)
	GetMessageCount(roomName string) (int64, error)
	
	// Search operations
	SearchMessages(query string, roomName string, limit int) ([]*Message, error)
}

// service implements Service interface
type service struct {
	repo Repository
}

// NewService creates a new message service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// SendMessage sends a new message
func (s *service) SendMessage(message *Message) error {
	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Validate message
	if err := s.validateMessage(message); err != nil {
		return err
	}

	// Save message
	if err := s.repo.SaveMessage(message); err != nil {
		return fmt.Errorf("failed to save message: %v", err)
	}

	log.Printf("💬 Message sent by %s in room %s: %s", message.Username, message.RoomName, message.Content)
	return nil
}

// GetMessage retrieves a message by ID
func (s *service) GetMessage(messageID string) (*Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message ID cannot be empty")
	}

	return s.repo.GetMessage(messageID)
}

// UpdateMessage updates an existing message
func (s *service) UpdateMessage(message *Message) error {
	if message.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	// Validate message
	if err := s.validateMessage(message); err != nil {
		return err
	}

	if err := s.repo.UpdateMessage(message); err != nil {
		return fmt.Errorf("failed to update message: %v", err)
	}

	log.Printf("✏️ Message updated by %s: %s", message.Username, message.Content)
	return nil
}

// DeleteMessage deletes a message
func (s *service) DeleteMessage(messageID string) error {
	if messageID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	if err := s.repo.DeleteMessage(messageID); err != nil {
		return fmt.Errorf("failed to delete message: %v", err)
	}

	log.Printf("🗑️ Message deleted: %s", messageID)
	return nil
}

// GetMessageHistory retrieves message history for a room
func (s *service) GetMessageHistory(roomName string, limit int) ([]*Message, error) {
	if roomName == "" {
		return nil, fmt.Errorf("room name cannot be empty")
	}

	return s.repo.GetMessageHistory(roomName, limit)
}

// GetRecentMessages retrieves recent messages across all rooms
func (s *service) GetRecentMessages(limit int) ([]*Message, error) {
	return s.repo.GetRecentMessages(limit)
}

// GetUserMessageHistory retrieves message history for a specific user
func (s *service) GetUserMessageHistory(username string, limit int) ([]*Message, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	return s.repo.GetUserMessageHistory(username, limit)
}

// GetMessageCount returns the total number of messages in a room
func (s *service) GetMessageCount(roomName string) (int64, error) {
	return s.repo.GetMessageCount(roomName)
}

// SearchMessages searches for messages containing specific text
func (s *service) SearchMessages(query string, roomName string, limit int) ([]*Message, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	return s.repo.SearchMessages(query, roomName, limit)
}

// validateMessage validates a message before saving
func (s *service) validateMessage(message *Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	if message.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	if message.Username == "" {
		return fmt.Errorf("message username cannot be empty")
	}

	if message.RoomName == "" {
		return fmt.Errorf("message room name cannot be empty")
	}

	if message.Type == "" {
		message.Type = "chat" // Default type
	}

	if message.Sender == "" {
		message.Sender = message.Username // Default sender
	}

	return nil
}