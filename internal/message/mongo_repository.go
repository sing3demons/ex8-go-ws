package message

import (
	"context"
	"fmt"
	"time"

	"realtime-chat/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository implements Repository interface using MongoDB
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository creates a new MongoDB message repository
func NewMongoRepository(db *database.MongoDB) Repository {
	return &MongoRepository{
		collection: db.GetCollection("messages"),
	}
}

// SaveMessage saves a message to MongoDB
func (r *MongoRepository) SaveMessage(message *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	messageDoc := &MessageDocument{
		Type:      message.Type,
		Content:   message.Content,
		Username:  message.Username,
		RoomName:  message.RoomName,
		Timestamp: message.Timestamp,
		Sender:    message.Sender,
		CreatedAt: now,
	}

	result, err := r.collection.InsertOne(ctx, messageDoc)
	if err != nil {
		return fmt.Errorf("failed to save message: %v", err)
	}

	// Update message with MongoDB ID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		message.ID = oid.Hex()
	}

	return nil
}

// GetMessage retrieves a single message by ID
func (r *MongoRepository) GetMessage(messageID string) (*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return nil, fmt.Errorf("invalid message ID: %v", err)
	}

	var messageDoc MessageDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&messageDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %v", err)
	}

	return messageDoc.ToMessage(), nil
}

// UpdateMessage updates an existing message
func (r *MongoRepository) UpdateMessage(message *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(message.ID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}

	update := bson.M{
		"$set": bson.M{
			"content":   message.Content,
			"type":      message.Type,
			"timestamp": message.Timestamp,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return fmt.Errorf("failed to update message: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// DeleteMessage deletes a message by ID
func (r *MongoRepository) DeleteMessage(messageID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return fmt.Errorf("invalid message ID: %v", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return fmt.Errorf("failed to delete message: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// GetMessageHistory retrieves message history for a room
func (r *MongoRepository) GetMessageHistory(roomName string, limit int) ([]*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set default limit if not specified
	if limit <= 0 {
		limit = 50
	}

	// Find messages for the room, sorted by timestamp descending
	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{"room_name": roomName}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve message history: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var messageDoc MessageDocument
		if err := cursor.Decode(&messageDoc); err != nil {
			continue
		}

		messages = append(messages, messageDoc.ToMessage())
	}

	// Reverse the slice to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetRecentMessages retrieves recent messages across all rooms
func (r *MongoRepository) GetRecentMessages(limit int) ([]*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve recent messages: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var messageDoc MessageDocument
		if err := cursor.Decode(&messageDoc); err != nil {
			continue
		}

		messages = append(messages, messageDoc.ToMessage())
	}

	return messages, nil
}

// GetUserMessageHistory retrieves message history for a specific user
func (r *MongoRepository) GetUserMessageHistory(username string, limit int) ([]*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{"username": username}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user message history: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var messageDoc MessageDocument
		if err := cursor.Decode(&messageDoc); err != nil {
			continue
		}

		messages = append(messages, messageDoc.ToMessage())
	}

	return messages, nil
}

// GetMessageCount returns the total number of messages in a room
func (r *MongoRepository) GetMessageCount(roomName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}
	if roomName != "" {
		filter["room_name"] = roomName
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %v", err)
	}

	return count, nil
}

// SearchMessages searches for messages containing specific text
func (r *MongoRepository) SearchMessages(query string, roomName string, limit int) ([]*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 50
	}

	// Build search filter
	filter := bson.M{
		"content": bson.M{"$regex": query, "$options": "i"}, // Case-insensitive search
	}

	if roomName != "" {
		filter["room_name"] = roomName
	}

	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var messageDoc MessageDocument
		if err := cursor.Decode(&messageDoc); err != nil {
			continue
		}

		messages = append(messages, messageDoc.ToMessage())
	}

	return messages, nil
}