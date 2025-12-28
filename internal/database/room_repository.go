package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"realtime-chat/internal/chat"
)

// MongoRoomRepository implements chat.RoomRepository using MongoDB
type MongoRoomRepository struct {
	collection     *mongo.Collection
	userRepository *MongoUserRepository
}

// NewMongoRoomRepository creates a new MongoDB room repository
func NewMongoRoomRepository(db *MongoDB, userRepo *MongoUserRepository) *MongoRoomRepository {
	return &MongoRoomRepository{
		collection:     db.GetCollection("rooms"),
		userRepository: userRepo,
	}
}

// Create creates a new room
func (r *MongoRoomRepository) Create(name, creatorUsername string, maxUsers int) (*chat.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if room already exists
	if _, exists := r.GetByName(name); exists {
		return nil, fmt.Errorf("room '%s' already exists", name)
	}

	now := time.Now()
	roomDoc := &RoomDocument{
		Name:        name,
		CreatedAt:   now,
		CreatedBy:   creatorUsername,
		MaxUsers:    maxUsers,
		IsActive:    true,
		UserCount:   0,
		LastMessage: now,
		UpdatedAt:   now,
	}

	result, err := r.collection.InsertOne(ctx, roomDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %v", err)
	}

	// Convert to chat.Room
	room := &chat.Room{
		Name:      name,
		Users:     make(map[string]*chat.User),
		CreatedAt: now,
		CreatedBy: creatorUsername,
		MaxUsers:  maxUsers,
		IsActive:  true,
	}

	_ = result.InsertedID // We have the ID but don't need it for the return

	return room, nil
}

// GetByName gets a room by name
func (r *MongoRoomRepository) GetByName(name string) (*chat.Room, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var roomDoc RoomDocument
	err := r.collection.FindOne(ctx, bson.M{"name": name, "is_active": true}).Decode(&roomDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false
		}
		return nil, false
	}

	// Get users in this room
	users := r.getUsersInRoom(name)
	userMap := make(map[string]*chat.User)
	for _, user := range users {
		userMap[user.ConnID] = user
	}

	room := &chat.Room{
		Name:      roomDoc.Name,
		Users:     userMap,
		CreatedAt: roomDoc.CreatedAt,
		CreatedBy: roomDoc.CreatedBy,
		MaxUsers:  roomDoc.MaxUsers,
		IsActive:  roomDoc.IsActive,
	}

	return room, true
}

// GetActiveRooms returns all active rooms
func (r *MongoRoomRepository) GetActiveRooms() []*chat.Room {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"is_active": true}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return []*chat.Room{}
	}
	defer cursor.Close(ctx)

	var rooms []*chat.Room
	for cursor.Next(ctx) {
		var roomDoc RoomDocument
		if err := cursor.Decode(&roomDoc); err != nil {
			continue
		}

		// Get users in this room
		users := r.getUsersInRoom(roomDoc.Name)
		userMap := make(map[string]*chat.User)
		for _, user := range users {
			userMap[user.ConnID] = user
		}

		room := &chat.Room{
			Name:      roomDoc.Name,
			Users:     userMap,
			CreatedAt: roomDoc.CreatedAt,
			CreatedBy: roomDoc.CreatedBy,
			MaxUsers:  roomDoc.MaxUsers,
			IsActive:  roomDoc.IsActive,
		}
		rooms = append(rooms, room)
	}

	return rooms
}

// GetAll returns all rooms (active and inactive)
func (r *MongoRoomRepository) GetAll() []*chat.Room {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return []*chat.Room{}
	}
	defer cursor.Close(ctx)

	var rooms []*chat.Room
	for cursor.Next(ctx) {
		var roomDoc RoomDocument
		if err := cursor.Decode(&roomDoc); err != nil {
			continue
		}

		// Get users in this room
		users := r.getUsersInRoom(roomDoc.Name)
		userMap := make(map[string]*chat.User)
		for _, user := range users {
			userMap[user.ConnID] = user
		}

		room := &chat.Room{
			Name:      roomDoc.Name,
			Users:     userMap,
			CreatedAt: roomDoc.CreatedAt,
			CreatedBy: roomDoc.CreatedBy,
			MaxUsers:  roomDoc.MaxUsers,
			IsActive:  roomDoc.IsActive,
		}
		rooms = append(rooms, room)
	}

	return rooms
}

// JoinRoom adds a user to a room
func (r *MongoRoomRepository) JoinRoom(user *chat.User, roomName string) error {
	// First, ensure the room exists
	room, exists := r.GetByName(roomName)
	if !exists {
		// Create default room if it doesn't exist
		if roomName == "general" {
			_, err := r.Create(roomName, "System", 100)
			if err != nil {
				return fmt.Errorf("failed to create default room: %v", err)
			}
		} else {
			return fmt.Errorf("room '%s' does not exist", roomName)
		}
	} else {
		// Check room capacity
		if len(room.Users) >= room.MaxUsers {
			return fmt.Errorf("room '%s' is full (%d/%d)", roomName, len(room.Users), room.MaxUsers)
		}
	}

	// Update user's current room
	if err := r.userRepository.UpdateCurrentRoom(user.ConnID, roomName); err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	// Update user object
	user.CurrentRoom = roomName

	// Update room's user count
	r.updateRoomUserCount(roomName)

	return nil
}

// LeaveRoom removes a user from a room
func (r *MongoRoomRepository) LeaveRoom(user *chat.User, roomName string) error {
	// Update user's current room to empty
	if err := r.userRepository.UpdateCurrentRoom(user.ConnID, ""); err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	// Update user object
	user.CurrentRoom = ""

	// Update room's user count
	r.updateRoomUserCount(roomName)

	return nil
}

// GetUsersInRoom returns all users in a specific room
func (r *MongoRoomRepository) GetUsersInRoom(roomName string) []*chat.User {
	return r.getUsersInRoom(roomName)
}

// GetRoomCount returns the number of active rooms
func (r *MongoRoomRepository) GetRoomCount() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"is_active": true})
	if err != nil {
		return 0
	}

	return int(count)
}

// getUsersInRoom is a helper method to get users in a room
func (r *MongoRoomRepository) getUsersInRoom(roomName string) []*chat.User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.userRepository.collection.Find(ctx, bson.M{
		"current_room":     roomName,
		"is_authenticated": true,
	})
	if err != nil {
		return []*chat.User{}
	}
	defer cursor.Close(ctx)

	var users []*chat.User
	for cursor.Next(ctx) {
		var userDoc UserDocument
		if err := cursor.Decode(&userDoc); err != nil {
			continue
		}

		user := &chat.User{
			ID:              userDoc.ID.Hex(),
			Username:        userDoc.Username,
			ConnID:          userDoc.ConnID,
			CurrentRoom:     userDoc.CurrentRoom,
			JoinedAt:        userDoc.JoinedAt,
			LastActive:      userDoc.LastActive,
			IsAuthenticated: userDoc.IsAuthenticated,
		}
		users = append(users, user)
	}

	return users
}

// updateRoomUserCount updates the user count for a room
func (r *MongoRoomRepository) updateRoomUserCount(roomName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count users in the room
	userCount, err := r.userRepository.collection.CountDocuments(ctx, bson.M{
		"current_room":     roomName,
		"is_authenticated": true,
	})
	if err != nil {
		return
	}

	// Update room document
	update := bson.M{
		"$set": bson.M{
			"user_count":   int(userCount),
			"last_message": time.Now(),
			"updated_at":   time.Now(),
		},
	}

	r.collection.UpdateOne(ctx, bson.M{"name": roomName}, update)
}

// DeactivateRoom deactivates a room
func (r *MongoRoomRepository) DeactivateRoom(roomName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"name": roomName}, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate room: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("room not found")
	}

	return nil
}