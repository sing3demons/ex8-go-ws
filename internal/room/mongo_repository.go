package room

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"realtime-chat/internal/database"
	userPkg "realtime-chat/internal/user"
)

// UserDocument represents a user document in MongoDB (for room queries)
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

// ToRoom converts RoomDocument to Room entity
func (doc *RoomDocument) ToRoom() *Room {
	return &Room{
		Name:      doc.Name,
		Users:     make(map[string]*userPkg.User), // Will be populated separately
		CreatedAt: doc.CreatedAt,
		CreatedBy: doc.CreatedBy,
		MaxUsers:  doc.MaxUsers,
		IsActive:  doc.IsActive,
	}
}

// FromRoom converts Room entity to RoomDocument
func (doc *RoomDocument) FromRoom(room *Room) {
	doc.Name = room.Name
	doc.CreatedAt = room.CreatedAt
	doc.CreatedBy = room.CreatedBy
	doc.MaxUsers = room.MaxUsers
	doc.IsActive = room.IsActive
	doc.UserCount = len(room.Users)
	doc.UpdatedAt = time.Now()
}

// MongoRepository implements Repository using MongoDB
type MongoRepository struct {
	collection     *mongo.Collection
	userCollection *mongo.Collection
}

// NewMongoRepository creates a new MongoDB room repository
func NewMongoRepository(db *database.MongoDB) Repository {
	return &MongoRepository{
		collection:     db.GetCollection("rooms"),
		userCollection: db.GetCollection("users"),
	}
}

// Create creates a new room
func (r *MongoRepository) Create(name, creatorUsername string, maxUsers int) (*Room, error) {
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

	// Convert to room.Room
	room := &Room{
		Name:      name,
		Users:     make(map[string]*userPkg.User),
		CreatedAt: now,
		CreatedBy: creatorUsername,
		MaxUsers:  maxUsers,
		IsActive:  true,
	}

	_ = result.InsertedID // We have the ID but don't need it for the return

	return room, nil
}

// GetByName gets a room by name
func (r *MongoRepository) GetByName(name string) (*Room, bool) {
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
	userMap := make(map[string]*userPkg.User)
	for _, user := range users {
		userMap[user.ConnID] = user
	}

	room := &Room{
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
func (r *MongoRepository) GetActiveRooms() []*Room {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"is_active": true}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return []*Room{}
	}
	defer cursor.Close(ctx)

	var rooms []*Room
	for cursor.Next(ctx) {
		var roomDoc RoomDocument
		if err := cursor.Decode(&roomDoc); err != nil {
			continue
		}

		// Get users in this room
		users := r.getUsersInRoom(roomDoc.Name)
		userMap := make(map[string]*userPkg.User)
		for _, user := range users {
			userMap[user.ConnID] = user
		}

		room := &Room{
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
func (r *MongoRepository) GetAll() []*Room {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return []*Room{}
	}
	defer cursor.Close(ctx)

	var rooms []*Room
	for cursor.Next(ctx) {
		var roomDoc RoomDocument
		if err := cursor.Decode(&roomDoc); err != nil {
			continue
		}

		// Get users in this room
		users := r.getUsersInRoom(roomDoc.Name)
		userMap := make(map[string]*userPkg.User)
		for _, user := range users {
			userMap[user.ConnID] = user
		}

		room := &Room{
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
func (r *MongoRepository) JoinRoom(user *userPkg.User, roomName string) error {
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

	// Update user's current room in database
	if err := r.updateUserCurrentRoom(user.ConnID, roomName); err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	// Update user object
	user.CurrentRoom = roomName

	// Update room's user count
	r.updateRoomUserCount(roomName)

	return nil
}

// LeaveRoom removes a user from a room
func (r *MongoRepository) LeaveRoom(user *userPkg.User, roomName string) error {
	// Update user's current room to empty in database
	if err := r.updateUserCurrentRoom(user.ConnID, ""); err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	// Update user object
	user.CurrentRoom = ""

	// Update room's user count
	r.updateRoomUserCount(roomName)

	return nil
}

// GetUsersInRoom returns all users in a specific room
func (r *MongoRepository) GetUsersInRoom(roomName string) []*userPkg.User {
	return r.getUsersInRoom(roomName)
}

// GetRoomCount returns the number of active rooms
func (r *MongoRepository) GetRoomCount() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"is_active": true})
	if err != nil {
		return 0
	}

	return int(count)
}

// getUsersInRoom is a helper method to get users in a room
func (r *MongoRepository) getUsersInRoom(roomName string) []*userPkg.User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.userCollection.Find(ctx, bson.M{
		"current_room":     roomName,
		"is_authenticated": true,
	})
	if err != nil {
		return []*userPkg.User{}
	}
	defer cursor.Close(ctx)

	var users []*userPkg.User
	for cursor.Next(ctx) {
		var userDoc UserDocument
		if err := cursor.Decode(&userDoc); err != nil {
			continue
		}

		user := &userPkg.User{
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

// updateUserCurrentRoom updates user's current room
func (r *MongoRepository) updateUserCurrentRoom(connID, roomName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"current_room": roomName,
			"updated_at":   time.Now(),
		},
	}

	result, err := r.userCollection.UpdateOne(ctx, bson.M{"conn_id": connID}, update)
	if err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// updateRoomUserCount updates the user count for a room
func (r *MongoRepository) updateRoomUserCount(roomName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count users in the room
	userCount, err := r.userCollection.CountDocuments(ctx, bson.M{
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
func (r *MongoRepository) DeactivateRoom(roomName string) error {
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