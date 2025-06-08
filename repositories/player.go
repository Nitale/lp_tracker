package repositories

import (
	"context"
	"fmt"
	"time"

	"lp_tracker/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerRepository struct {
	collection *mongo.Collection
}

func NewPlayerRepository(db *mongo.Database) *PlayerRepository {
	collection := db.Collection("players")

	// Create indexes for better performance
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "gameName", Value: 1},
			{Key: "tagLine", Value: 1},
			{Key: "server", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// Create index (ignore error if already exists)
	collection.Indexes().CreateOne(context.Background(), indexModel)

	return &PlayerRepository{
		collection: collection,
	}
}

// Create adds a new player to the database
func (r *PlayerRepository) Create(ctx context.Context, player *models.Player) error {
	player.CreatedAt = time.Now()
	player.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, player)
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}

	// Set the ID from the result
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		player.ID = oid
	}

	return nil
}

// FindByRiotID finds a player by their Riot ID (gameName + tagLine + server)
func (r *PlayerRepository) FindByRiotID(ctx context.Context, gameName, tagLine, server string) (*models.Player, error) {
	var player models.Player

	filter := bson.M{
		"gameName": gameName,
		"tagLine":  tagLine,
		"server":   server,
	}

	err := r.collection.FindOne(ctx, filter).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Player not found, return nil instead of error
		}
		return nil, fmt.Errorf("failed to find player: %w", err)
	}

	return &player, nil
}

// FindByPUUID finds a player by their PUUID
func (r *PlayerRepository) FindByPUUID(ctx context.Context, puuid string) (*models.Player, error) {
	var player models.Player

	err := r.collection.FindOne(ctx, bson.M{"puuid": puuid}).Decode(&player)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find player by PUUID: %w", err)
	}

	return &player, nil
}

// Update updates an existing player
func (r *PlayerRepository) Update(ctx context.Context, player *models.Player) error {
	player.UpdatedAt = time.Now()

	filter := bson.M{"_id": player.ID}
	_, err := r.collection.ReplaceOne(ctx, filter, player)
	if err != nil {
		return fmt.Errorf("failed to update player: %w", err)
	}

	return nil
}

// FindAll returns all players in the database
func (r *PlayerRepository) FindAll(ctx context.Context) ([]*models.Player, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find players: %w", err)
	}
	defer cursor.Close(ctx)

	var players []*models.Player
	for cursor.Next(ctx) {
		var player models.Player
		if err := cursor.Decode(&player); err != nil {
			return nil, fmt.Errorf("failed to decode player: %w", err)
		}
		players = append(players, &player)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return players, nil
}

// FindAllWithPagination returns players with pagination
func (r *PlayerRepository) FindAllWithPagination(ctx context.Context, page, limit int) ([]*models.Player, int64, error) {
	// Count total documents
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count players: %w", err)
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find with pagination
	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find players: %w", err)
	}
	defer cursor.Close(ctx)

	var players []*models.Player
	for cursor.Next(ctx) {
		var player models.Player
		if err := cursor.Decode(&player); err != nil {
			return nil, 0, fmt.Errorf("failed to decode player: %w", err)
		}
		players = append(players, &player)
	}

	return players, total, nil
}

// Delete removes a player from the database
func (r *PlayerRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}

	return nil
}

// DeleteByRiotID removes a player by their Riot ID
func (r *PlayerRepository) DeleteByRiotID(ctx context.Context, gameName, tagLine, server string) error {
	filter := bson.M{
		"gameName": gameName,
		"tagLine":  tagLine,
		"server":   server,
	}

	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}

	return nil
}

// Exists checks if a player exists
func (r *PlayerRepository) Exists(ctx context.Context, gameName, tagLine, server string) (bool, error) {
	filter := bson.M{
		"gameName": gameName,
		"tagLine":  tagLine,
		"server":   server,
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check player existence: %w", err)
	}

	return count > 0, nil
}

// FindByServer returns all players from a specific server
func (r *PlayerRepository) FindByServer(ctx context.Context, server string) ([]*models.Player, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"server": server})
	if err != nil {
		return nil, fmt.Errorf("failed to find players by server: %w", err)
	}
	defer cursor.Close(ctx)

	var players []*models.Player
	for cursor.Next(ctx) {
		var player models.Player
		if err := cursor.Decode(&player); err != nil {
			return nil, fmt.Errorf("failed to decode player: %w", err)
		}
		players = append(players, &player)
	}

	return players, nil
}
