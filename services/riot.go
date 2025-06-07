package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lp_tracker/models"
)

type RiotService struct {
	apiKey     string
	httpClient *http.Client
}

// Riot API response structures
type AccountDTO struct {
	PUUID    string `json:"puuid"`
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
}

type SummonerDTO struct {
	ID            string `json:"id"`
	AccountID     string `json:"accountId"`
	PUUID         string `json:"puuid"`
	Name          string `json:"name"`
	ProfileIconID int    `json:"profileIconId"`
	RevisionDate  int64  `json:"revisionDate"`
	SummonerLevel int    `json:"summonerLevel"`
}

type LeagueEntryDTO struct {
	LeagueID     string `json:"leagueId"`
	SummonerID   string `json:"summonerId"`
	SummonerName string `json:"summonerName"`
	QueueType    string `json:"queueType"`
	Tier         string `json:"tier"`
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	HotStreak    bool   `json:"hotStreak"`
	Veteran      bool   `json:"veteran"`
	FreshBlood   bool   `json:"freshBlood"`
	Inactive     bool   `json:"inactive"`
}

func NewRiotService(apiKey string) *RiotService {
	if apiKey == "" {
		panic("Riot API key is required")
	}

	return &RiotService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *RiotService) GetPlayerByRiotID(ctx context.Context, gameName, tagLine, server string) (*models.Player, error) {
	// Step 1: Get account by Riot ID
	account, err := r.getAccountByRiotID(ctx, gameName, tagLine)
	if err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	// Step 2: Get summoner by PUUID
	summoner, err := r.getSummonerByPUUID(ctx, account.PUUID, server)
	if err != nil {
		return nil, fmt.Errorf("failed to get summoner: %w", err)
	}

	// Step 3: Get ranked information
	var tierStr, rankStr string
	var leaguePoints, wins, losses int

	leagueEntries, err := r.getLeagueEntriesBySummonerID(ctx, summoner.ID, server)
	if err != nil {
		// Continue with default values (unranked)
		tierStr = "UNRANKED"
		rankStr = ""
		leaguePoints = 0
		wins = 0
		losses = 0
	} else {
		// Find Solo/Duo ranked queue
		rankedEntry := r.findRankedSoloEntry(leagueEntries)
		if rankedEntry != nil {
			tierStr = rankedEntry.Tier
			rankStr = rankedEntry.Rank
			leaguePoints = rankedEntry.LeaguePoints
			wins = rankedEntry.Wins
			losses = rankedEntry.Losses
		} else {
			tierStr = "UNRANKED"
			rankStr = ""
			leaguePoints = 0
			wins = 0
			losses = 0
		}
	}

	player := &models.Player{
		PUUID:         account.PUUID,
		GameName:      gameName,
		TagLine:       tagLine,
		Server:        server,
		SummonerLevel: summoner.SummonerLevel,
		Tier:          tierStr,
		Rank:          rankStr,
		LeaguePoints:  leaguePoints,
		Wins:          wins,
		Losses:        losses,
	}

	return player, nil
}

func (r *RiotService) UpdatePlayerRank(ctx context.Context, player *models.Player) error {
	// Get summoner by PUUID
	summoner, err := r.getSummonerByPUUID(ctx, player.PUUID, player.Server)
	if err != nil {
		return fmt.Errorf("failed to get summoner: %w", err)
	}

	// Get updated league entries
	leagueEntries, err := r.getLeagueEntriesBySummonerID(ctx, summoner.ID, player.Server)
	if err != nil {
		return fmt.Errorf("failed to get league entries: %w", err)
	}

	// Find Solo/Duo ranked queue
	rankedEntry := r.findRankedSoloEntry(leagueEntries)
	if rankedEntry != nil {
		player.Tier = rankedEntry.Tier
		player.Rank = rankedEntry.Rank
		player.LeaguePoints = rankedEntry.LeaguePoints
		player.Wins = rankedEntry.Wins
		player.Losses = rankedEntry.Losses
	} else {
		player.Tier = "UNRANKED"
		player.Rank = ""
		player.LeaguePoints = 0
		player.Wins = 0
		player.Losses = 0
	}

	player.SummonerLevel = summoner.SummonerLevel
	return nil
}

// Helper methods for direct API calls

func (r *RiotService) getAccountByRiotID(ctx context.Context, gameName, tagLine string) (*AccountDTO, error) {
	url := fmt.Sprintf("https://europe.api.riotgames.com/riot/account/v1/accounts/by-riot-id/%s/%s", gameName, tagLine)
	
	var account AccountDTO
	err := r.makeAPIRequest(ctx, url, &account)
	if err != nil {
		return nil, err
	}
	
	return &account, nil
}

func (r *RiotService) getSummonerByPUUID(ctx context.Context, puuid, server string) (*SummonerDTO, error) {
	baseURL, err := r.getAPIBaseURL(server)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/lol/summoner/v4/summoners/by-puuid/%s", baseURL, puuid)
	
	var summoner SummonerDTO
	err = r.makeAPIRequest(ctx, url, &summoner)
	if err != nil {
		return nil, err
	}
	
	return &summoner, nil
}

func (r *RiotService) getLeagueEntriesBySummonerID(ctx context.Context, summonerID, server string) ([]LeagueEntryDTO, error) {
	baseURL, err := r.getAPIBaseURL(server)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/lol/league/v4/entries/by-summoner/%s", baseURL, summonerID)
	
	var entries []LeagueEntryDTO
	err = r.makeAPIRequest(ctx, url, &entries)
	if err != nil {
		return nil, err
	}
	
	return entries, nil
}

func (r *RiotService) makeAPIRequest(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("X-Riot-Token", r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(body, target)
}

func (r *RiotService) getAPIBaseURL(server string) (string, error) {
	server = strings.ToLower(server)
	
	switch server {
	case "euw1", "euw":
		return "https://euw1.api.riotgames.com", nil
	case "eun1", "eune":
		return "https://eun1.api.riotgames.com", nil
	case "na1", "na":
		return "https://na1.api.riotgames.com", nil
	case "kr":
		return "https://kr.api.riotgames.com", nil
	case "jp1", "jp":
		return "https://jp1.api.riotgames.com", nil
	case "br1", "br":
		return "https://br1.api.riotgames.com", nil
	case "la1", "lan":
		return "https://la1.api.riotgames.com", nil
	case "la2", "las":
		return "https://la2.api.riotgames.com", nil
	case "oc1", "oce":
		return "https://oc1.api.riotgames.com", nil
	case "tr1", "tr":
		return "https://tr1.api.riotgames.com", nil
	case "ru":
		return "https://ru.api.riotgames.com", nil
	default:
		return "", fmt.Errorf("unsupported server: %s", server)
	}
}

func (r *RiotService) findRankedSoloEntry(entries []LeagueEntryDTO) *LeagueEntryDTO {
	for _, entry := range entries {
		if entry.QueueType == "RANKED_SOLO_5x5" {
			return &entry
		}
	}
	return nil
}