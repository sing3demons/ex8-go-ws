package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"realtime-chat/internal/chat"
)

// MongoUserRepository implements chat.UserRepository using MongoDB
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new MongoDB user repository
func NewMongoUserRepository(db *MongoDB) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.GetCollection("users"),
	}
}

// Create creates a new user
func (r *MongoUserRepository) Create(connID, username string) (*chat.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if username already exists
	if !r.IsUsernameAvailable(username) {
		return nil, fmt.Errorf("username '%s' is already taken", username)
	}

	now := time.Now()
	userDoc := &UserDocument{
		Username:        username,
		ConnID:          connID,
		CurrentRoom:     "",
		JoinedAt:        now,
		LastActive:      now,
		IsAuthenticated: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	result, err := r.collection.InsertOne(ctx, userDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// Convert to chat.User
	user := &chat.User{
		ID:              result.InsertedID.(primitive.ObjectID).Hex(),
		Username:        username,
		ConnID:          connID,
		CurrentRoom:     "",
		JoinedAt:        now,
		LastActive:      now,
		IsAuthenticated: true,
	}

	return user, nil
}

// GetByID gets a user by connection ID
func (r *MongoUserRepository) GetByID(connID string) (*chat.User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userDoc UserDocument
	err := r.collection.FindOne(ctx, bson.M{"conn_id": connID}).Decode(&userDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false
		}
		return nil, false
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

	return user, true
}

// GetByUsername gets a user by username
func (r *MongoUserRepository) GetByUsername(username string) (*chat.User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userDoc UserDocument
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&userDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false
		}
		return nil, false
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

	return user, true
}

// IsUsernameAvailable checks if a username is available
func (r *MongoUserRepository) IsUsernameAvailable(username string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"username": username})
	if err != nil {
		return false
	}

	return count == 0
}

// GetAll returns all users
func (r *MongoUserRepository) GetAll() []*chat.User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
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

// UpdateLastActive updates user's last active time
func (r *MongoUserRepository) UpdateLastActive(connID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"last_active": time.Now(),
			"updated_at":  time.Now(),
		},
	}

	r.collection.UpdateOne(ctx, bson.M{"conn_id": connID}, update)
}

// UpdateCurrentRoom updates user's current room
func (r *MongoUserRepository) UpdateCurrentRoom(connID, roomName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"current_room": roomName,
			"updated_at":   time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"conn_id": connID}, update)
	if err != nil {
		return fmt.Errorf("failed to update user room: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete removes a user
func (r *MongoUserRepository) Delete(connID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"conn_id": connID})
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetActiveUsers returns users active within the specified duration
func (r *MongoUserRepository) GetActiveUsers(since time.Duration) []*chat.User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-since)
	filter := bson.M{
		"last_active": bson.M{"$gte": cutoff},
		"is_authenticated": true,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"last_active": -1}))
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