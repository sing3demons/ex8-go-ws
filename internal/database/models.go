package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoConfig holds MongoDB configuration
type MongoConfig struct {
	URI            string        `json:"uri"`
	Database       string        `json:"database"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	PingTimeout    time.Duration `json:"ping_timeout"`
	MaxPoolSize    uint64        `json:"max_pool_size"`
	MinPoolSize    uint64        `json:"min_pool_size"`
}

// DefaultMongoConfig returns default MongoDB configuration
func DefaultMongoConfig() *MongoConfig {
	return &MongoConfig{
		URI:            "mongodb://localhost:27017",
		Database:       "realtime_chat",
		ConnectTimeout: 10 * time.Second,
		PingTimeout:    5 * time.Second,
		MaxPoolSize:    100,
		MinPoolSize:    5,
	}
}

// MongoDB represents a MongoDB connection
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	config   *MongoConfig
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(config *MongoConfig) (*MongoDB, error) {
	if config == nil {
		config = DefaultMongoConfig()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetConnectTimeout(config.ConnectTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Test the connection
	pingCtx, pingCancel := context.WithTimeout(context.Background(), config.PingTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := &MongoDB{
		client:   client,
		database: client.Database(config.Database),
		config:   config,
	}

	log.Printf("✅ Connected to MongoDB: %s/%s", config.URI, config.Database)
	return db, nil
}

// GetCollection returns a MongoDB collection
func (db *MongoDB) GetCollection(name string) *mongo.Collection {
	return db.database.Collection(name)
}

// Close closes the MongoDB connection
func (db *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %v", err)
	}

	log.Println("✅ Disconnected from MongoDB")
	return nil
}

// CreateIndexes creates necessary indexes for collections
func (db *MongoDB) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// User indexes
	userCollection := db.GetCollection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "conn_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	if _, err := userCollection.Indexes().CreateMany(ctx, userIndexes); err != nil {
		return fmt.Errorf("failed to create user indexes: %v", err)
	}

	// Room indexes
	roomCollection := db.GetCollection("rooms")
	roomIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	}

	if _, err := roomCollection.Indexes().CreateMany(ctx, roomIndexes); err != nil {
		return fmt.Errorf("failed to create room indexes: %v", err)
	}

	// Message indexes
	messageCollection := db.GetCollection("messages")
	messageIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "room_name", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "username", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "content", Value: "text"}},
		},
	}

	if _, err := messageCollection.Indexes().CreateMany(ctx, messageIndexes); err != nil {
		return fmt.Errorf("failed to create message indexes: %v", err)
	}

	log.Println("✅ MongoDB indexes created successfully")
	return nil
}
// GetDatabase returns the database instance
func (db *MongoDB) GetDatabase() *mongo.Database {
	return db.database
}

// GetClient returns the client instance
func (db *MongoDB) GetClient() *mongo.Client {
	return db.client
}

// HealthCheck performs a health check on the MongoDB connection
func (db *MongoDB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), db.config.PingTimeout)
	defer cancel()

	if err := db.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("MongoDB health check failed: %v", err)
	}

	return nil
}