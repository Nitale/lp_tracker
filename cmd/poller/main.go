package main

import (
	"context"
	"fmt"
	"log"
	"lp_tracker/database"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if os.Getenv("DOCKER_ENV") != "true" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	}

	// Validate required environment variables
	requiredEnvs := map[string]string{
		"MONGO_URI":      os.Getenv("MONGO_LOCAL_URI"),
		"MONGO_DATABASE": os.Getenv("MONGO_DATABASE"),
	}
	fmt.Println(requiredEnvs)

	for key, value := range requiredEnvs {
		if value == "" {
			log.Fatalf("%s environment variable is required", key)
		}
	}

	ctx := context.Background()

	// MongoDB connection
	dbConfig := database.Config{
		URI:          os.Getenv("MONGO_URI"),
		DatabaseName: os.Getenv("MONGO_DATABASE"),
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
