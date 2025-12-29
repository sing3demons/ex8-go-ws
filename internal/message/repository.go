package message

// Repository interface for message persistence
type Repository interface {
	// Basic message operations
	SaveMessage(message *Message) error
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

// EnhancedRepository interface for enhanced message operations
type EnhancedRepository interface {
	// Basic CRUD operations
	SaveEnhancedMessage(message *EnhancedMessage) error
	GetEnhancedMessage(messageID string) (*EnhancedMessage, error)
	UpdateEnhancedMessage(message *EnhancedMessage) error
	DeleteEnhancedMessage(messageID string) error
	
	// Reaction operations
	AddReaction(messageID string, reaction MessageReaction) error
	RemoveReaction(messageID, emoji, username string) error
	GetMessageReactions(messageID string) ([]MessageReaction, error)
	
	// Thread operations
	GetThreadMessages(threadID string) ([]*EnhancedMessage, error)
	GetMessageThread(messageID string) ([]*EnhancedMessage, error)
	UpdateThreadCount(parentMessageID string) error
	
	// Search operations
	SearchEnhancedMessages(query SearchQuery) ([]*EnhancedMessage, error)
	GetMessagesByFilter(filter MessageFilter) ([]*EnhancedMessage, error)
	
	// Status operations
	UpdateMessageStatus(messageID string, status MessageStatus) error
	GetUnreadMessagesByUser(username string) ([]*EnhancedMessage, error)
	MarkMessagesAsRead(messageIDs []string, username string) error
	
	// Analytics operations
	GetMessageStats(roomName string, period string) (*MessageStats, error)
	GetUserMessageCount(username string, period string) (int64, error)
	GetPopularReactions(roomName string, limit int) ([]ReactionStats, error)
}