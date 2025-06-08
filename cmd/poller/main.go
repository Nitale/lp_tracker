package main

import (
	"context"
	"log"
	"lp_tracker/database"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	ctx := context.Background()

	mongoURI := os.Getenv("MONGO_URI")
	mongoDatabase := os.Getenv("MONGO_DATABASE")

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
