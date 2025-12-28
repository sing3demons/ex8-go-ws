package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserDocument represents a user document in MongoDB
type UserDocument struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username        string             `bson:"username" json:"username"`
	ConnID          string             `bson:"conn_id" json:"conn_id"`
	CurrentRoom     string             `bson:"current_room" json:"current_room"`
	JoinedAt        time.Time          `bson:"joined_at" json:"joined_at"`
	LastActive      time.Time          `bson:"last_active" json:"last_active"`
	IsAuthenticated bool               `bson:"is_authenticated" json:"is_authenticated"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// RoomDocument represents a room document in MongoDB
type RoomDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	MaxUsers    int                `bson:"max_users" json:"max_users"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	UserCount   int                `bson:"user_count" json:"user_count"`
	LastMessage time.Time          `bson:"last_message" json:"last_message"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// MessageDocument represents a message document in MongoDB
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