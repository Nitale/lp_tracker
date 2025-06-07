package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lp_tracker/database"
	"lp_tracker/discord"
	"lp_tracker/container"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Validate required environment variables
	requiredEnvs := map[string]string{
		"DISCORD_TOKEN": os.Getenv("DISCORD_TOKEN"),
		"RIOT_API_KEY":  os.Getenv("RIOT_API_KEY"),
		"MONGO_URI": os.Getenv("MONGO_URI"),
		"MONGO_DATABASE": os.Getenv("MONGO_DATABASE"),
	}

	for key, value := range requiredEnvs {
		if value == "" {
			log.Fatalf("%s environment variable is required", key)
		}
	}

	// Database configuration
	dbConfig := database.Config{
		URI:          os.Getenv("MONGO_URI"),
		DatabaseName: os.Getenv("MONGO_DATABASE"),
		Timeout:      30 * time.Second,
	}

	// Initialize database manager
	dbManager, err := database.NewManager(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		dbManager.Close(ctx)
	}()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = dbManager.Ping(ctx)
	if err != nil {
		log.Fatal("Database health check failed:", err)
	}
	log.Println("Database health check passed!")

	// Initialize service container
	serviceContainer := container.NewContainer(dbManager, os.Getenv("RIOT_API_KEY"))

	// Create Discord session
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Initialize command handler with service container
	commandHandler := discord.NewCommandHandler(serviceContainer)

	// Add handlers
	dg.AddHandler(commandHandler.HandleInteraction)
	
	// Register commands AFTER connection is established
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Printf("Bot is ready and serving %d guilds", len(s.State.Guilds))
		
		// Now register slash commands (bot is connected)
		log.Println("Registering slash commands...")
		err := commandHandler.RegisterCommands(s)
		if err != nil {
			log.Printf("Error registering commands: %v", err)
		} else {
			log.Println("✅ Slash commands registered successfully!")
		}
	})

	// Set intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open connection
	log.Println("🔄 Connecting to Discord...")
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening Discord connection:", err)
	}
	defer dg.Close()

	log.Println("🤖 Discord bot is running! Press CTRL+C to exit.")

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down Discord bot...")
	
	// Cleanup
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Close database connection
	err = dbManager.Close(ctx)
	if err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
	
	log.Println("✅ Shutdown complete")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}