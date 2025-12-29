package analytics

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SessionDocument represents a user session in MongoDB
type SessionDocument struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ConnID       string             `bson:"conn_id" json:"conn_id"`
	Username     string             `bson:"username" json:"username"`
	StartTime    time.Time          `bson:"start_time" json:"start_time"`
	EndTime      *time.Time         `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Duration     time.Duration      `bson:"duration" json:"duration"`
	MessageCount int                `bson:"message_count" json:"message_count"`
	RoomsVisited []string           `bson:"rooms_visited" json:"rooms_visited"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// RoomStatsDocument represents room statistics in MongoDB
type RoomStatsDocument struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoomName      string             `bson:"room_name" json:"room_name"`
	Date          time.Time          `bson:"date" json:"date"`
	MessageCount  int                `bson:"message_count" json:"message_count"`
	UniqueUsers   int                `bson:"unique_users" json:"unique_users"`
	PeakUsers     int                `bson:"peak_users" json:"peak_users"`
	TotalDuration time.Duration      `bson:"total_duration" json:"total_duration"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// Session represents a user session (domain entity)
type Session struct {
	ID           string        `json:"id"`
	ConnID       string        `json:"conn_id"`
	Username     string        `json:"username"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      *time.Time    `json:"end_time,omitempty"`
	Duration     time.Duration `json:"duration"`
	MessageCount int           `json:"message_count"`
	RoomsVisited []string      `json:"rooms_visited"`
	IsActive     bool          `json:"is_active"`
}

// RoomStats represents room statistics (domain entity)
type RoomStats struct {
	ID            string        `json:"id"`
	RoomName      string        `json:"room_name"`
	Date          time.Time     `json:"date"`
	MessageCount  int           `json:"message_count"`
	UniqueUsers   int           `json:"unique_users"`
	PeakUsers     int           `json:"peak_users"`
	TotalDuration time.Duration `json:"total_duration"`
}

// ToSession converts SessionDocument to Session entity
func (doc *SessionDocument) ToSession() *Session {
	return &Session{
		ID:           doc.ID.Hex(),
		ConnID:       doc.ConnID,
		Username:     doc.Username,
		StartTime:    doc.StartTime,
		EndTime:      doc.EndTime,
		Duration:     doc.Duration,
		MessageCount: doc.MessageCount,
		RoomsVisited: doc.RoomsVisited,
		IsActive:     doc.IsActive,
	}
}

// FromSession converts Session entity to SessionDocument
func (doc *SessionDocument) FromSession(session *Session) {
	doc.ConnID = session.ConnID
	doc.Username = session.Username
	doc.StartTime = session.StartTime
	doc.EndTime = session.EndTime
	doc.Duration = session.Duration
	doc.MessageCount = session.MessageCount
	doc.RoomsVisited = session.RoomsVisited
	doc.IsActive = session.IsActive
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if session.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(session.ID); err == nil {
			doc.ID = oid
		}
	}
}

// ToRoomStats converts RoomStatsDocument to RoomStats entity
func (doc *RoomStatsDocument) ToRoomStats() *RoomStats {
	return &RoomStats{
		ID:            doc.ID.Hex(),
		RoomName:      doc.RoomName,
		Date:          doc.Date,
		MessageCount:  doc.MessageCount,
		UniqueUsers:   doc.UniqueUsers,
		PeakUsers:     doc.PeakUsers,
		TotalDuration: doc.TotalDuration,
	}
}

// FromRoomStats converts RoomStats entity to RoomStatsDocument
func (doc *RoomStatsDocument) FromRoomStats(stats *RoomStats) {
	doc.RoomName = stats.RoomName
	doc.Date = stats.Date
	doc.MessageCount = stats.MessageCount
	doc.UniqueUsers = stats.UniqueUsers
	doc.PeakUsers = stats.PeakUsers
	doc.TotalDuration = stats.TotalDuration
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if stats.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(stats.ID); err == nil {
			doc.ID = oid
		}
	}
}