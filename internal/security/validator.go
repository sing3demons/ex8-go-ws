package security

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"realtime-chat/internal/config"
)

// InputValidator handles input validation and sanitization
type InputValidator struct {
	config *config.ServerConfig
}

// NewInputValidator creates a new input validator
func NewInputValidator(config *config.ServerConfig) *InputValidator {
	return &InputValidator{
		config: config,
	}
}

// ValidateUsername validates and sanitizes username input
func (v *InputValidator) ValidateUsername(username string) (string, error) {
	// Trim whitespace
	username = strings.TrimSpace(username)
	
	// Check if empty
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}
	
	// Check length
	if utf8.RuneCountInString(username) > v.config.MaxUsernameLength {
		return "", fmt.Errorf("username too long (max %d characters)", v.config.MaxUsernameLength)
	}
	
	// Check for invalid characters (only allow alphanumeric, underscore, hyphen, and Thai characters)
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_\-\p{Thai}]+$`)
	if !validUsername.MatchString(username) {
		return "", fmt.Errorf("username contains invalid characters (only letters, numbers, _, - allowed)")
	}
	
	// Sanitize HTML
	username = html.EscapeString(username)
	
	return username, nil
}

// ValidateMessage validates and sanitizes message content
func (v *InputValidator) ValidateMessage(message string) (string, error) {
	// Check if empty
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("message cannot be empty")
	}
	
	// Check length
	if utf8.RuneCountInString(message) > v.config.MaxMessageLength {
		return "", fmt.Errorf("message too long (max %d characters)", v.config.MaxMessageLength)
	}
	
	// Remove excessive whitespace
	message = strings.TrimSpace(message)
	message = regexp.MustCompile(`\s+`).ReplaceAllString(message, " ")
	
	// Sanitize HTML to prevent XSS
	message = html.EscapeString(message)
	
	// Check for spam patterns (repeated characters)
	if v.isSpamMessage(message) {
		return "", fmt.Errorf("message appears to be spam")
	}
	
	return message, nil
}

// ValidateRoomName validates and sanitizes room name
func (v *InputValidator) ValidateRoomName(roomName string) (string, error) {
	// Trim whitespace
	roomName = strings.TrimSpace(roomName)
	
	// Check if empty
	if roomName == "" {
		return "", fmt.Errorf("room name cannot be empty")
	}
	
	// Check length
	if utf8.RuneCountInString(roomName) > v.config.MaxRoomNameLength {
		return "", fmt.Errorf("room name too long (max %d characters)", v.config.MaxRoomNameLength)
	}
	
	// Check for invalid characters (no spaces, only alphanumeric, underscore, hyphen)
	validRoomName := regexp.MustCompile(`^[a-zA-Z0-9_\-\p{Thai}]+$`)
	if !validRoomName.MatchString(roomName) {
		return "", fmt.Errorf("room name contains invalid characters (no spaces, only letters, numbers, _, - allowed)")
	}
	
	// Convert to lowercase for consistency
	roomName = strings.ToLower(roomName)
	
	// Sanitize HTML
	roomName = html.EscapeString(roomName)
	
	return roomName, nil
}

// isSpamMessage checks if a message appears to be spam
func (v *InputValidator) isSpamMessage(message string) bool {
	// Check for excessive repeated characters (simple approach)
	if len(message) > 20 {
		charCount := make(map[rune]int)
		for _, char := range message {
			charCount[char]++
			if charCount[char] > 10 {
				return true
			}
		}
	}
	
	// Check for excessive repeated words
	words := strings.Fields(message)
	if len(words) > 5 {
		wordCount := make(map[string]int)
		for _, word := range words {
			wordCount[strings.ToLower(word)]++
			if wordCount[strings.ToLower(word)] > 3 {
				return true
			}
		}
	}
	
	return false
}

// SanitizeHTML removes or escapes HTML tags from input
func (v *InputValidator) SanitizeHTML(input string) string {
	return html.EscapeString(input)
}

// ValidateCommand validates command input
func (v *InputValidator) ValidateCommand(command string) (string, error) {
	// Trim whitespace
	command = strings.TrimSpace(command)
	
	// Check if it starts with /
	if !strings.HasPrefix(command, "/") {
		return "", fmt.Errorf("command must start with /")
	}
	
	// Check length (commands should be shorter)
	if utf8.RuneCountInString(command) > 100 {
		return "", fmt.Errorf("command too long")
	}
	
	// Sanitize HTML
	command = html.EscapeString(command)
	
	return command, nil
}