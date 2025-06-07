package container

import (
	"lp_tracker/database"
	"lp_tracker/repositories"
	"lp_tracker/services"
)

// Container holds all application dependencies
type Container struct {
	// Database
	DB *database.Manager

	// Repositories
	PlayerRepo *repositories.PlayerRepository

	// Services
	PlayerService *services.PlayerService
	RiotService   *services.RiotService
}

// NewContainer creates and initializes all dependencies
func NewContainer(dbManager *database.Manager, riotAPIKey string) *Container {
	// Initialize repositories
	playerRepo := repositories.NewPlayerRepository(dbManager.GetDatabase())

	// Initialize services
	riotService := services.NewRiotService(riotAPIKey)
	playerService := services.NewPlayerService(playerRepo, riotAPIKey)

	return &Container{
		DB:            dbManager,
		PlayerRepo:    playerRepo,
		PlayerService: playerService,
		RiotService:   riotService,
	}
}

// GetPlayerService returns the player service
func (c *Container) GetPlayerService() *services.PlayerService {
	return c.PlayerService
}

// GetRiotService returns the riot service
func (c *Container) GetRiotService() *services.RiotService {
	return c.RiotService
}

// GetPlayerRepository returns the player repository
func (c *Container) GetPlayerRepository() *repositories.PlayerRepository {
	return c.PlayerRepo
}