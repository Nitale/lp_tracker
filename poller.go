package main

import (
	"context"
	"os"
	"fmt"
	"log"
	"lp_tracker/database"
	"github.com/joho/godotenv"
	"time"
)

func main() {
	err := godotenv.Load()
    if err != nil {
        log.Printf("Warning: Error loading .env file: %v", err)
    }

    ctx := context.Background()

	mongoUser := os.Getenv("MONGO_ROOT_USERNAME")
    mongoPassword := os.Getenv("MONGO_ROOT_PASSWORD")
    mongoDatabase := os.Getenv("MONGO_DATABASE")

	mongoURI := fmt.Sprintf("mongodb://%s:%s@localhost:27017/%s?authSource=admin", mongoUser, mongoPassword, mongoDatabase)

	// MongoDB connection
	dbConfig := database.Config{
		URI:          mongoURI,
		DatabaseName: mongoDatabase,
		Timeout:      30 * time.Second,
	}
	
	dbManager, err := database.NewManager(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	
	err = dbManager.Ping(ctx)
    if err != nil {
        log.Printf("Failed to ping database: %v", err)
    } else {
        log.Println("Database ping successful!")
    }
}