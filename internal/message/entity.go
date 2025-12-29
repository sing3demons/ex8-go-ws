package message

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a basic message (backward compatibility)
type Message struct {
	ID        string    `json:"id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Username  string    `json:"username"`
	RoomName  string    `json:"room_name"`
	Timestamp time.Time `json:"timestamp"`
}

// EnhancedMessage represents an enhanced message with additional features
type EnhancedMessage struct {
	// Existing fields (backward compatibility)
	ID        string    `json:"id" bson:"_id,omitempty"`
	Type      string    `json:"type" bson:"type"`
	Content   string    `json:"content" bson:"content"`
	Username  string    `json:"username" bson:"username"`
	RoomName  string    `json:"room_name" bson:"room_name"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Sender    string    `json:"sender" bson:"sender"`

	// New enhanced fields
	ParentID     *string              `json:"parent_id,omitempty" bson:"parent_id,omitempty"`
	ThreadID     *string              `json:"thread_id,omitempty" bson:"thread_id,omitempty"`
	Reactions    []MessageReaction    `json:"reactions,omitempty" bson:"reactions,omitempty"`
	Attachments  []MessageAttachment  `json:"attachments,omitempty" bson:"attachments,omitempty"`
	Mentions     []string             `json:"mentions,omitempty" bson:"mentions,omitempty"`
	EditHistory  []MessageEdit        `json:"edit_history,omitempty" bson:"edit_history,omitempty"`
	Status       MessageStatus        `json:"status" bson:"status"`
	Formatting   MessageFormatting    `json:"formatting,omitempty" bson:"formatting,omitempty"`
	IsDeleted    bool                 `json:"is_deleted" bson:"is_deleted"`
	DeletedAt    *time.Time           `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
	CreatedAt    time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at" bson:"updated_at"`
}

// MessageReaction represents a reaction to a message
type MessageReaction struct {
	Emoji     string    `json:"emoji" bson:"emoji"`
	Users     []string  `json:"users" bson:"users"`
	Count     int       `json:"count" bson:"count"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// MessageAttachment represents a file attachment to a message
type MessageAttachment struct {
	ID           string    `json:"id" bson:"id"`
	FileName     string    `json:"file_name" bson:"file_name"`
	FileSize     int64     `json:"file_size" bson:"file_size"`
	FileType     string    `json:"file_type" bson:"file_type"`
	MimeType     string    `json:"mime_type" bson:"mime_type"`
	URL          string    `json:"url" bson:"url"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty" bson:"thumbnail_url,omitempty"`
	UploadedAt   time.Time `json:"uploaded_at" bson:"uploaded_at"`
}

// MessageEdit represents an edit to a message
type MessageEdit struct {
	PreviousContent string    `json:"previous_content" bson:"previous_content"`
	EditedAt        time.Time `json:"edited_at" bson:"edited_at"`
	EditReason      string    `json:"edit_reason,omitempty" bson:"edit_reason,omitempty"`
}

// MessageStatus represents the delivery and read status of a message
type MessageStatus struct {
	Sent      time.Time     `json:"sent" bson:"sent"`
	Delivered *time.Time    `json:"delivered,omitempty" bson:"delivered,omitempty"`
	ReadBy    []ReadReceipt `json:"read_by,omitempty" bson:"read_by,omitempty"`
}

// ReadReceipt represents when a user read a message
type ReadReceipt struct {
	Username string    `json:"username" bson:"username"`
	ReadAt   time.Time `json:"read_at" bson:"read_at"`
}

// MessageFormatting represents formatting information for rich text
type MessageFormatting struct {
	IsMarkdown   bool     `json:"is_markdown" bson:"is_markdown"`
	HasCodeBlock bool     `json:"has_code_block" bson:"has_code_block"`
	Language     *string  `json:"language,omitempty" bson:"language,omitempty"`
	Links        []string `json:"links,omitempty" bson:"links,omitempty"`
}

// Thread represents a message thread
type Thread struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
	ParentID     string    `json:"parent_id" bson:"parent_id"`
	RoomName     string    `json:"room_name" bson:"room_name"`
	CreatedBy    string    `json:"created_by" bson:"created_by"`
	ReplyCount   int       `json:"reply_count" bson:"reply_count"`
	LastReplyAt  time.Time `json:"last_reply_at" bson:"last_reply_at"`
	Participants []string  `json:"participants" bson:"participants"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

// SearchQuery represents a search query for messages
type SearchQuery struct {
	Text          string     `json:"text"`
	Username      string     `json:"username,omitempty"`
	RoomName      string     `json:"room_name,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	MessageType   string     `json:"message_type,omitempty"`
	HasAttachment bool       `json:"has_attachment"`
	Limit         int        `json:"limit"`
	Offset        int        `json:"offset"`
}

// MessageFilter represents filtering criteria for messages
type MessageFilter struct {
	RoomName      string     `json:"room_name,omitempty"`
	Username      string     `json:"username,omitempty"`
	MessageType   string     `json:"message_type,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	HasReactions  bool       `json:"has_reactions"`
	HasAttachment bool       `json:"has_attachment"`
	IsThread      bool       `json:"is_thread"`
	IsDeleted     bool       `json:"is_deleted"`
	Limit         int        `json:"limit"`
	Offset        int        `json:"offset"`
}

// MessageStats represents analytics data for messages
type MessageStats struct {
	TotalMessages    int64                    `json:"total_messages"`
	MessagesByType   map[string]int64         `json:"messages_by_type"`
	MessagesByUser   map[string]int64         `json:"messages_by_user"`
	PopularReactions []ReactionStats          `json:"popular_reactions"`
	PeakHours        map[int]int64            `json:"peak_hours"`
	ThreadCount      int64                    `json:"thread_count"`
	AttachmentCount  int64                    `json:"attachment_count"`
	Period           string                   `json:"period"`
	GeneratedAt      time.Time                `json:"generated_at"`
}

// ReactionStats represents statistics for reactions
type ReactionStats struct {
	Emoji string `json:"emoji"`
	Count int64  `json:"count"`
	Users int64  `json:"users"`
}

// Notification represents a user notification
type Notification struct {
	ID          string                 `json:"id" bson:"_id,omitempty"`
	UserID      string                 `json:"user_id" bson:"user_id"`
	Type        string                 `json:"type" bson:"type"`
	Title       string                 `json:"title" bson:"title"`
	Message     string                 `json:"message" bson:"message"`
	Data        map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`
	IsRead      bool                   `json:"is_read" bson:"is_read"`
	ReadAt      *time.Time             `json:"read_at,omitempty" bson:"read_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
}

// NotificationTypes defines the types of notifications
const (
	NotificationTypeMention      = "mention"
	NotificationTypeReply        = "reply"
	NotificationTypeReaction     = "reaction"
	NotificationTypeEdit         = "edit"
	NotificationTypeModeration   = "moderation"
	NotificationTypeSystem       = "system"
)

// MessageDocument represents the MongoDB document structure for basic messages
type MessageDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      string             `bson:"type" json:"type"`
	Content   string             `bson:"content" json:"content"`
	Username  string             `bson:"username" json:"username"`
	RoomName  string             `bson:"room_name" json:"room_name"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	Sender    string             `bson:"sender" json:"sender"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// EnhancedMessageDocument represents the MongoDB document structure for enhanced messages
type EnhancedMessageDocument struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty"`
	Type         string               `bson:"type"`
	Content      string               `bson:"content"`
	Username     string               `bson:"username"`
	RoomName     string               `bson:"room_name"`
	Timestamp    time.Time            `bson:"timestamp"`
	Sender       string               `bson:"sender"`
	ParentID     *primitive.ObjectID  `bson:"parent_id,omitempty"`
	ThreadID     *primitive.ObjectID  `bson:"thread_id,omitempty"`
	Reactions    []MessageReaction    `bson:"reactions,omitempty"`
	Attachments  []MessageAttachment  `bson:"attachments,omitempty"`
	Mentions     []string             `bson:"mentions,omitempty"`
	EditHistory  []MessageEdit        `bson:"edit_history,omitempty"`
	Status       MessageStatus        `bson:"status"`
	Formatting   MessageFormatting    `bson:"formatting,omitempty"`
	IsDeleted    bool                 `bson:"is_deleted"`
	DeletedAt    *time.Time           `bson:"deleted_at,omitempty"`
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
}

// ThreadDocument represents the MongoDB document structure for threads
type ThreadDocument struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	ParentID     primitive.ObjectID `bson:"parent_id"`
	RoomName     string             `bson:"room_name"`
	CreatedBy    string             `bson:"created_by"`
	ReplyCount   int                `bson:"reply_count"`
	LastReplyAt  time.Time          `bson:"last_reply_at"`
	Participants []string           `bson:"participants"`
	CreatedAt    time.Time          `bson:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at"`
}

// NotificationDocument represents the MongoDB document structure for notifications
type NotificationDocument struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty"`
	UserID    string                 `bson:"user_id"`
	Type      string                 `bson:"type"`
	Title     string                 `bson:"title"`
	Message   string                 `bson:"message"`
	Data      map[string]interface{} `bson:"data,omitempty"`
	IsRead    bool                   `bson:"is_read"`
	ReadAt    *time.Time             `bson:"read_at,omitempty"`
	CreatedAt time.Time              `bson:"created_at"`
	ExpiresAt *time.Time             `bson:"expires_at,omitempty"`
}

// Helper methods for backward compatibility

// ToEnhancedMessage converts a basic Message to EnhancedMessage
func (m *Message) ToEnhancedMessage() *EnhancedMessage {
	now := time.Now()
	return &EnhancedMessage{
		ID:        m.ID,
		Type:      m.Type,
		Content:   m.Content,
		Username:  m.Username,
		RoomName:  m.RoomName,
		Timestamp: m.Timestamp,
		Sender:    m.Sender,
		Status: MessageStatus{
			Sent: now,
		},
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// FromEnhancedMessage converts an EnhancedMessage to basic Message
func (m *Message) FromEnhancedMessage(enhanced *EnhancedMessage) {
	m.ID = enhanced.ID
	m.Type = enhanced.Type
	m.Content = enhanced.Content
	m.Username = enhanced.Username
	m.RoomName = enhanced.RoomName
	m.Timestamp = enhanced.Timestamp
	m.Sender = enhanced.Sender
}

// ToMessage converts MessageDocument to basic Message (backward compatibility)
func (doc *MessageDocument) ToMessage() *Message {
	return &Message{
		ID:        doc.ID.Hex(),
		Type:      doc.Type,
		Content:   doc.Content,
		Username:  doc.Username,
		RoomName:  doc.RoomName,
		Timestamp: doc.Timestamp,
		Sender:    doc.Sender,
	}
}

// FromMessage converts basic Message to MessageDocument
func (doc *MessageDocument) FromMessage(msg *Message) {
	doc.Type = msg.Type
	doc.Content = msg.Content
	doc.Username = msg.Username
	doc.RoomName = msg.RoomName
	doc.Timestamp = msg.Timestamp
	doc.Sender = msg.Sender
	doc.CreatedAt = time.Now()

	if msg.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(msg.ID); err == nil {
			doc.ID = oid
		}
	}
}