package discord

import (
	"context"
	"fmt"
	"log"
	"strings"

	"lp_tracker/container"
	"lp_tracker/models"
	"lp_tracker/services"

	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CommandHandler struct {
	container     *container.Container
	playerService *services.PlayerService
	workerPool    chan struct{}
	stats         *CommandStats
}

type CommandStats struct {
	mu             sync.Mutex
	totalCommands  int64
	activeCommands int64
	averageTime    time.Duration
}

func NewCommandHandler(c *container.Container) *CommandHandler {
	return &CommandHandler{
		container:     c,
		playerService: c.GetPlayerService(),
		// worker pool limit to 2 to avoid overwhelming riot api (since poller which also poll Riot API runs in parallel)
		workerPool: make(chan struct{}, 2),
		stats:      &CommandStats{},
	}
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "add_player",
		Description: "Add a player to the tracking database",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "pseudo",
				Description: "Player's game name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tagline",
				Description: "Player's tagline (without #)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "server",
				Description: "Server region",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "EUW (Europe West)", Value: "euw1"},
					{Name: "EUNE (Europe Nordic & East)", Value: "eun1"},
					{Name: "NA (North America)", Value: "na1"},
					{Name: "KR (Korea)", Value: "kr"},
					{Name: "JP (Japan)", Value: "jp1"},
					{Name: "BR (Brazil)", Value: "br1"},
					{Name: "LAN (Latin America North)", Value: "la1"},
					{Name: "LAS (Latin America South)", Value: "la2"},
					{Name: "OCE (Oceania)", Value: "oc1"},
					{Name: "TR (Turkey)", Value: "tr1"},
					{Name: "RU (Russia)", Value: "ru"},
				},
			},
		},
	},
	{
		Name:        "list_players",
		Description: "List all tracked players",
	},
}

func (h *CommandHandler) RegisterCommands(s *discordgo.Session) error {
	log.Println("Registering slash commands...")

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("failed to create command %s: %v", cmd.Name, err)
		}
		log.Printf("Registered command: %s", cmd.Name)
	}

	return nil
}

func (h *CommandHandler) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// i.ApplicationCommandData().Name is an implicit routine (Discordgo)
	switch i.ApplicationCommandData().Name {
	case "add_player":
		go h.handleAddPlayerAsync(s, i)
	case "list_players":
		go h.handleListPlayersAsync(s, i)
	}
}

func (h *CommandHandler) handleAddPlayerAsync(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//Add a worker to the pool (similar as a ticket in a queue) - We use struct{}{} because we don't need to store any data (optimization)
	h.workerPool <- struct{}{}
	//Remove from the pool when the function returns
	defer func() { <-h.workerPool }()

	// Statistics
	start := time.Now()
	h.updateStats(1, 0)
	defer func() {
		h.updateStats(-1, time.Since(start))
	}()

	// Defer response to avoid timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring response: %v", err)
		return
	}

	log.Printf("🔄 Starting processAddPlayer for user interaction")
	h.processAddPlayer(s, i)
}

func (h *CommandHandler) processAddPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	pseudo := options[0].StringValue()
	tagline := options[1].StringValue()
	server := strings.ToLower(options[2].StringValue())

	// context with Timeout to avoid hanging API requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type result struct {
		player *models.Player
		err    error
	}
	resultChan := make(chan result, 1)

	// Add Player in a goroutine
	go func() {
		player, err := h.playerService.AddPlayer(ctx, pseudo, tagline, server)
		resultChan <- result{player: player, err: err}
	}()

	//Wait for Timeout or result
	select {
	case res := <-resultChan:
		if res.err != nil {
			h.handleAddPlayerErrors(s, i, res.err, pseudo, tagline, server)
			return
		}
		log.Printf("✅ AddPlayer success, sending success message")
		h.sendAppPlayerSuccess(s, i, res.player)
	case <-ctx.Done():
		h.sendFollowUp(s, i, "❌ Request timed out. Please try again later.")
		log.Printf("Add player timed out: %s#%s on server %s", pseudo, tagline, server)
	}
}

func (h *CommandHandler) handleListPlayersAsync(s *discordgo.Session, i *discordgo.InteractionCreate) {
	h.workerPool <- struct{}{}
	defer func() { <-h.workerPool }()

	// Statistics
	start := time.Now()
	h.updateStats(1, 0)
	defer func() {
		h.updateStats(-1, time.Since(start))
	}()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring response: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parallelize the request to the database
	playersChan := make(chan []*models.Player, 1)
	errorChan := make(chan error, 1)

	go func() {
		players, err := h.playerService.GetAllPlayers(ctx)
		if err != nil {
			errorChan <- err
			return
		}
		playersChan <- players
	}()

	select {
	case players := <-playersChan:
		h.sendPlayersList(s, i, players)
	case err := <-errorChan:
		h.sendFollowUp(s, i, fmt.Sprintf("❌ Failed to fetch players from database: %v", err))
		log.Printf("Error fetching players from database: %v", err)
	case <-ctx.Done():
		h.sendFollowUp(s, i, "❌ Request timed out")
	}
}

func (h *CommandHandler) handleAddPlayerErrors(s *discordgo.Session, i *discordgo.InteractionCreate, err error, pseudo string, tagline string, server string) {
	var response string
	if strings.Contains(err.Error(), "already being tracked") {
		response = fmt.Sprintf("❌ Player **%s#%s** (%s) is already being tracked!", pseudo, tagline, strings.ToUpper(server))
	} else if strings.Contains(err.Error(), "not found") {
		response = fmt.Sprintf("❌ Player **%s#%s** not found on server **%s**\n\n💡 **Tips:**\n• Check the spelling of the name and tagline\n• Make sure the server is correct\n• The player might not exist or have never played ranked",
			pseudo, tagline, strings.ToUpper(server))
	} else {
		response = fmt.Sprintf("❌ Failed to add player **%s#%s**\n\n**Error:** %v", pseudo, tagline, err)
	}
	h.sendFollowUp(s, i, response)
}

func (h *CommandHandler) sendAppPlayerSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, player *models.Player) {
	var rankInfo string
	if player.Tier == "UNRANKED" {
		rankInfo = "🆕 **Unranked**"
	} else {
		rankInfo = fmt.Sprintf("🏆 **%s %s** • %d LP", player.Tier, player.Rank, player.LeaguePoints)
	}

	response := fmt.Sprintf("✅ Successfully added **%s#%s** (%s)\n📊 **Level:** %d\n%s",
		player.GameName,
		player.TagLine,
		strings.ToUpper(player.Server),
		player.SummonerLevel,
		rankInfo,
	)
	h.sendFollowUp(s, i, response)
}

func (h *CommandHandler) sendPlayersList(s *discordgo.Session, i *discordgo.InteractionCreate, players []*models.Player) {
	if len(players) == 0 {
		h.sendFollowUp(s, i, "📭 No players tracked yet!\nUse `/add_player` to start tracking.")
		return
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📋 **Tracked Players (%d)**\n\n", len(players)))

	for idx, player := range players {
		if idx >= 20 {
			response.WriteString(fmt.Sprintf("... and %d more players\n", len(players)-20))
			break
		}

		var rankInfo string
		if player.Tier == "UNRANKED" {
			rankInfo = "🆕 Unranked"
		} else {
			rankInfo = fmt.Sprintf("🏆 %s %s %d LP", player.Tier, player.Rank, player.LeaguePoints)
		}

		response.WriteString(fmt.Sprintf("👤 **%s#%s** (%s)\n   📊 Level %d • %s\n\n",
			player.GameName, player.TagLine, strings.ToUpper(player.Server),
			player.SummonerLevel, rankInfo))
	}

	h.sendFollowUp(s, i, response.String())
}

func (h *CommandHandler) updateStats(delta int64, duration time.Duration) {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()

	if delta > 0 {
		h.stats.totalCommands++
		h.stats.activeCommands++
	} else {
		h.stats.activeCommands--
	}

	if duration > 0 {
		h.stats.averageTime = (h.stats.averageTime + duration) / 2
	}
}

func (h *CommandHandler) GetStats() (total int64, active int64, avgTime time.Duration) {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	return h.stats.totalCommands, h.stats.activeCommands, h.stats.averageTime
}

func (h *CommandHandler) sendFollowUp(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
	})
	if err != nil {
		log.Printf("Error sending followup message: %v", err)
	}
}
