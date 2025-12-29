package user

import (
	"context"
	"fmt"
	"realtime-chat/internal/database"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// ToUser converts UserDocument to User entity
func (doc *UserDocument) ToUser() *User {
	return &User{
		ID:              doc.ID.Hex(),
		Username:        doc.Username,
		ConnID:          doc.ConnID,
		CurrentRoom:     doc.CurrentRoom,
		JoinedAt:        doc.JoinedAt,
		LastActive:      doc.LastActive,
		IsAuthenticated: doc.IsAuthenticated,
	}
}

// FromUser converts User entity to UserDocument
func (doc *UserDocument) FromUser(user *User) {
	doc.Username = user.Username
	doc.ConnID = user.ConnID
	doc.CurrentRoom = user.CurrentRoom
	doc.JoinedAt = user.JoinedAt
	doc.LastActive = user.LastActive
	doc.IsAuthenticated = user.IsAuthenticated
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	if user.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(user.ID); err == nil {
			doc.ID = oid
		}
	}
}

// MongoRepository implements Repository using MongoDB
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository creates a new MongoDB user repository
func NewMongoRepository(db *database.MongoDB) Repository {
	return &MongoRepository{
		collection: db.GetCollection("users"),
	}
}

// Create creates a new user
func (r *MongoRepository) Create(connID, username string) (*User, error) {
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

	// Convert to user.User
	user := &User{
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
func (r *MongoRepository) GetByID(connID string) (*User, bool) {
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

	user := &User{
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
func (r *MongoRepository) GetByUsername(username string) (*User, bool) {
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

	user := &User{
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
func (r *MongoRepository) IsUsernameAvailable(username string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"username": username})
	if err != nil {
		return false
	}

	return count == 0
}

// GetAll returns all users
func (r *MongoRepository) GetAll() []*User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return []*User{}
	}
	defer cursor.Close(ctx)

	var users []*User
	for cursor.Next(ctx) {
		var userDoc UserDocument
		if err := cursor.Decode(&userDoc); err != nil {
			continue
		}

		user := &User{
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
func (r *MongoRepository) UpdateLastActive(connID string) {
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
func (r *MongoRepository) UpdateCurrentRoom(connID, roomName string) error {
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
func (r *MongoRepository) Delete(connID string) error {
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
func (r *MongoRepository) GetActiveUsers(since time.Duration) []*User {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-since)
	filter := bson.M{
		"last_active":      bson.M{"$gte": cutoff},
		"is_authenticated": true,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"last_active": -1}))
	if err != nil {
		return []*User{}
	}
	defer cursor.Close(ctx)

	var users []*User
	for cursor.Next(ctx) {
		var userDoc UserDocument
		if err := cursor.Decode(&userDoc); err != nil {
			continue
		}

		user := &User{
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
