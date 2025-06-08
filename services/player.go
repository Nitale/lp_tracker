package services

import (
	"context"
	"fmt"
	"time"

	"lp_tracker/models"
	"lp_tracker/repositories"
)

type PlayerService struct {
	playerRepo  *repositories.PlayerRepository
	riotService *RiotService
}

func NewPlayerService(playerRepo *repositories.PlayerRepository, riotAPIKey string) *PlayerService {
	return &PlayerService{
		playerRepo:  playerRepo,
		riotService: NewRiotService(riotAPIKey),
	}
}

// AddPlayer adds a new player to tracking
func (ps *PlayerService) AddPlayer(ctx context.Context, gameName, tagLine, server string) (*models.Player, error) {
	// Check if player already exists
	existingPlayer, err := ps.playerRepo.FindByRiotID(ctx, gameName, tagLine, server)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing player: %w", err)
	}

	if existingPlayer != nil {
		return nil, fmt.Errorf("player %s#%s (%s) is already being tracked", gameName, tagLine, server)
	}

	// Fetch player data from Riot API
	player, err := ps.riotService.GetPlayerByRiotID(ctx, gameName, tagLine, server)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player from Riot API: %w", err)
	}

	// Save player to database
	err = ps.playerRepo.Create(ctx, player)
	if err != nil {
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	return player, nil
}

// GetAllPlayers returns all tracked players
func (ps *PlayerService) GetAllPlayers(ctx context.Context) ([]*models.Player, error) {
	return ps.playerRepo.FindAll(ctx)
}

// GetPlayerByRiotID finds a player by their Riot ID
func (ps *PlayerService) GetPlayerByRiotID(ctx context.Context, gameName, tagLine, server string) (*models.Player, error) {
	return ps.playerRepo.FindByRiotID(ctx, gameName, tagLine, server)
}

// UpdatePlayer updates a single player's information
func (ps *PlayerService) UpdatePlayer(ctx context.Context, player *models.Player) error {
	// Update player data from Riot API
	err := ps.riotService.UpdatePlayerRank(ctx, player)
	if err != nil {
		return fmt.Errorf("failed to update player rank: %w", err)
	}

	// Save updated player to database
	err = ps.playerRepo.Update(ctx, player)
	if err != nil {
		return fmt.Errorf("failed to save updated player: %w", err)
	}

	return nil
}

// UpdateAllPlayers updates all tracked players' information
func (ps *PlayerService) UpdateAllPlayers(ctx context.Context) error {
	players, err := ps.playerRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch players: %w", err)
	}

	var errors []string
	for _, player := range players {
		err := ps.UpdatePlayer(ctx, player)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to update player %s#%s: %v", player.GameName, player.TagLine, err)
			errors = append(errors, errorMsg)
			fmt.Println(errorMsg)
			continue
		}

		// Rate limiting: wait between API calls
		time.Sleep(1 * time.Second)
	}

	if len(errors) > 0 {
		return fmt.Errorf("some players failed to update: %v", errors)
	}

	return nil
}
