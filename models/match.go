// models/match.go
package models

import (
	"fmt"
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MatchPlayerInfo represents the information of a match for a specific player
type MatchPlayerInfo struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	
	// Links to other collections
	PlayerPUUID string `bson:"player_puuid" json:"player_puuid"`       // Reference to the player
	MatchID     string `bson:"match_id" json:"match_id"`               // Riot Match ID
	
	// Basic player information
	Pseudo string `bson:"pseudo" json:"pseudo"`
	
	// Match result
	Victory bool `bson:"victory" json:"victory"`
	
	// Rank information (at the time of the match)
	Rank         string `bson:"rank" json:"rank"`                     // ex: "GOLD III"
	LeaguePoints int    `bson:"league_points" json:"league_points"`
	QueueType    string `bson:"queue_type" json:"queue_type"`         // "RANKED_SOLO_5x5" or "RANKED_FLEX_SR"
	
	// Player performance
	Kills     int    `bson:"kills" json:"kills"`
	Deaths    int    `bson:"deaths" json:"deaths"`
	Assists   int    `bson:"assists" json:"assists"`
	Champion  string `bson:"champion" json:"champion"`
	
	// Advanced statistics
	DamageToChamps int `bson:"damage_to_champs" json:"damage_to_champs"`
	CreepScore     int `bson:"creep_score" json:"creep_score"`           // CS total
	GoldEarned     int `bson:"gold_earned" json:"gold_earned"`
	VisionScore    int `bson:"vision_score" json:"vision_score"`
	
	// Metadata
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	ProcessedAt   time.Time `bson:"processed_at" json:"processed_at"`     // When this match was processed by the bot
	NotifiedAt    *time.Time `bson:"notified_at,omitempty" json:"notified_at,omitempty"` // When the Discord message was sent
}

// Useful methods for MatchPlayerInfo

// KDA calculation
func (m *MatchPlayerInfo) KDA() float64 {
	if m.Deaths == 0 {
		return float64(m.Kills + m.Assists)
	}
	return float64(m.Kills+m.Assists) / float64(m.Deaths)
}

// KDAString returns the KDA formatted as string
func (m *MatchPlayerInfo) KDAString() string {
	return fmt.Sprintf("%d/%d/%d", m.Kills, m.Deaths, m.Assists)
}

// FormatGameDuration returns the match duration formatted (MM:SS)
// func (m *MatchPlayerInfo) FormatGameDuration() string {
// 	minutes := m.GameDuration / 60
// 	seconds := m.GameDuration % 60
// 	return fmt.Sprintf("%d:%02d", minutes, seconds)
// }

// IsRanked checks if the match is ranked
func (m *MatchPlayerInfo) IsRanked() bool {
	return m.QueueType == "RANKED_SOLO_5x5" || m.QueueType == "RANKED_FLEX_SR"
}
