package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

type Manager struct {
	client   *mongo.Client
	database *mongo.Database
}

type Config struct {
	URI          string
	DatabaseName string
	Timeout      time.Duration
}

func NewManager(config Config) (*Manager, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Set client options
	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(100).
		SetMaxConnIdleTime(30 * time.Second).
		SetConnectTimeout(config.Timeout)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.DatabaseName)

	manager := &Manager{
		client:   client,
		database: database,
	}

	// Create indexes
	err = manager.createIndexes(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	log.Printf("Successfully connected to MongoDB database: %s", config.DatabaseName)
	return manager, nil
}

func (m *Manager) GetDatabase() *mongo.Database {
	return m.database
}

func (m *Manager) GetClient() *mongo.Client {
	return m.client
}

func (m *Manager) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}

func (m *Manager) createIndexes(ctx context.Context) error {
	// Create indexes for players collection
	playersCollection := m.database.Collection("players")
	
	playerIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "gameName", Value: 1},
				{Key: "tagLine", Value: 1},
				{Key: "server", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "puuid", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "server", Value: 1},
			},
		},
	}

	_, err := playersCollection.Indexes().CreateMany(ctx, playerIndexes)
	if err != nil {
		return fmt.Errorf("failed to create player indexes: %w", err)
	}

	log.Println("Successfully created database indexes")
	return nil
}