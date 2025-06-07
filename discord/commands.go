package discord

import (
	"context"
	"fmt"
	"log"
	"strings"

	"lp_tracker/container"
	"lp_tracker/services"

	"github.com/bwmarrin/discordgo"
)

type CommandHandler struct {
	container     *container.Container
	playerService *services.PlayerService
}

func NewCommandHandler(c *container.Container) *CommandHandler {
	return &CommandHandler{
		container:     c,
		playerService: c.GetPlayerService(),
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
	switch i.ApplicationCommandData().Name {
	case "add_player":
		h.handleAddPlayer(s, i)
	case "list_players":
		h.handleListPlayers(s, i)
	}
}

func (h *CommandHandler) handleAddPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer response to avoid timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring response: %v", err)
		return
	}

	options := i.ApplicationCommandData().Options
	pseudo := options[0].StringValue()
	tagline := options[1].StringValue()
	server := strings.ToLower(options[2].StringValue())

	// Validate inputs
	if len(pseudo) == 0 || len(tagline) == 0 {
		h.sendFollowUp(s, i, "❌ Player name and tagline cannot be empty!")
		return
	}

	ctx := context.Background()

	// Add player using service
	player, err := h.playerService.AddPlayer(ctx, pseudo, tagline, server)
	if err != nil {
		var response string
		if strings.Contains(err.Error(), "already being tracked") {
			// Get existing player info for better response
			existingPlayer, getErr := h.playerService.GetPlayerByRiotID(ctx, pseudo, tagline, server)
			if getErr == nil {
				response = fmt.Sprintf("❌ Player **%s#%s** (%s) is already being tracked!\n🏆 Current: %s %s %d LP", 
					pseudo, tagline, strings.ToUpper(server),
					existingPlayer.Tier, existingPlayer.Rank, existingPlayer.LeaguePoints)
			} else {
				response = fmt.Sprintf("❌ Player **%s#%s** (%s) is already being tracked!", pseudo, tagline, strings.ToUpper(server))
			}
		} else if strings.Contains(err.Error(), "not found") {
			response = fmt.Sprintf("❌ Player **%s#%s** not found on server **%s**\n\n💡 **Tips:**\n• Check the spelling of the name and tagline\n• Make sure the server is correct\n• The player might not exist or have never played ranked", 
				pseudo, tagline, strings.ToUpper(server))
		} else {
			response = fmt.Sprintf("❌ Failed to add player **%s#%s**\n\n**Error:** %v", pseudo, tagline, err)
		}
		h.sendFollowUp(s, i, response)
		return
	}

	// Success response with better formatting
	var rankInfo string
	if player.Tier == "UNRANKED" {
		rankInfo = "🆕 **Unranked**"
	} else {
		rankInfo = fmt.Sprintf("🏆 **%s %s** • %d LP", player.Tier, player.Rank, player.LeaguePoints)
	}

	response := fmt.Sprintf("✅ Successfully added **%s#%s** (%s)\n📊 **Level:** %d\n%s\n📈 **W/L:** %d/%d", 
		player.GameName, 
		player.TagLine, 
		strings.ToUpper(player.Server),
		player.SummonerLevel,
		rankInfo,
		player.Wins,
		player.Losses,
	)

	h.sendFollowUp(s, i, response)
}

func (h *CommandHandler) handleListPlayers(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring response: %v", err)
		return
	}

	ctx := context.Background()
	players, err := h.playerService.GetAllPlayers(ctx)
	if err != nil {
		h.sendFollowUp(s, i, "❌ Failed to fetch players from database")
		return
	}

	if len(players) == 0 {
		h.sendFollowUp(s, i, "📭 No players are currently being tracked.\nUse `/add_player` to start tracking a player!")
		return
	}

	// Build response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("📋 **Tracked Players (%d)**\n\n", len(players)))

	for i, player := range players {
		if i >= 20 { // Limit to prevent message being too long
			response.WriteString(fmt.Sprintf("... and %d more players\n", len(players)-20))
			break
		}

		var rankInfo string
		if player.Tier == "UNRANKED" {
			rankInfo = "🆕 Unranked"
		} else {
			rankInfo = fmt.Sprintf("🏆 %s %s %d LP", player.Tier, player.Rank, player.LeaguePoints)
		}

		response.WriteString(fmt.Sprintf("👤 **%s#%s** (%s)\n", 
			player.GameName, 
			player.TagLine, 
			strings.ToUpper(player.Server),
		))
		response.WriteString(fmt.Sprintf("   📊 Level %d • %s\n\n", 
			player.SummonerLevel,
			rankInfo,
		))
	}

	h.sendFollowUp(s, i, response.String())
}

func (h *CommandHandler) sendFollowUp(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
	})
	if err != nil {
		log.Printf("Error sending followup message: %v", err)
	}
}