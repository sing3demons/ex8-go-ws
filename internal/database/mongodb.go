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

// MongoDB represents a MongoDB connection
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	config   *MongoConfig
}

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

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(config *MongoConfig) (*MongoDB, error) {
	if config == nil {
		config = DefaultMongoConfig()
	}

	// Set client options
	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetConnectTimeout(config.ConnectTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	pingCtx, pingCancel := context.WithTimeout(context.Background(), config.PingTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	database := client.Database(config.Database)

	log.Printf("✅ Connected to MongoDB: %s/%s", config.URI, config.Database)

	return &MongoDB{
		client:   client,
		database: database,
		config:   config,
	}, nil
}

// GetDatabase returns the database instance
func (m *MongoDB) GetDatabase() *mongo.Database {
	return m.database
}

// GetClient returns the client instance
func (m *MongoDB) GetClient() *mongo.Client {
	return m.client
}

// GetCollection returns a collection
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Close closes the MongoDB connection
func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %v", err)
	}

	log.Println("✅ Disconnected from MongoDB")
	return nil
}

// CreateIndexes creates necessary indexes for collections
func (m *MongoDB) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Users collection indexes
	usersCollection := m.GetCollection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"username", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"conn_id", 1}},
		},
		{
			Keys: bson.D{{"last_active", -1}},
		},
	}

	if _, err := usersCollection.Indexes().CreateMany(ctx, userIndexes); err != nil {
		return fmt.Errorf("failed to create user indexes: %v", err)
	}

	// Rooms collection indexes
	roomsCollection := m.GetCollection("rooms")
	roomIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"name", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
		{
			Keys: bson.D{{"is_active", 1}},
		},
	}

	if _, err := roomsCollection.Indexes().CreateMany(ctx, roomIndexes); err != nil {
		return fmt.Errorf("failed to create room indexes: %v", err)
	}

	// Messages collection indexes
	messagesCollection := m.GetCollection("messages")
	messageIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"room_name", 1}, {"timestamp", -1}},
		},
		{
			Keys: bson.D{{"username", 1}, {"timestamp", -1}},
		},
		{
			Keys: bson.D{{"timestamp", -1}},
		},
	}

	if _, err := messagesCollection.Indexes().CreateMany(ctx, messageIndexes); err != nil {
		return fmt.Errorf("failed to create message indexes: %v", err)
	}

	log.Println("✅ Created MongoDB indexes")
	return nil
}

// HealthCheck performs a health check on the MongoDB connection
func (m *MongoDB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.PingTimeout)
	defer cancel()

	if err := m.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("MongoDB health check failed: %v", err)
	}

	return nil
}