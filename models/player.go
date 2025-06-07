package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Player struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PUUID         string             `bson:"puuid" json:"puuid"`
	GameName      string             `bson:"gameName" json:"gameName"`
	TagLine       string             `bson:"tagLine" json:"tagLine"`
	Server        string             `bson:"server" json:"server"`
	SummonerID    string             `bson:"summonerId" json:"summonerId"`
	SummonerLevel int                `bson:"summonerLevel" json:"summonerLevel"`
	ProfileIconID int                `bson:"profileIconId" json:"profileIconId"`
	
	// Ranked information
	Tier         string `bson:"tier" json:"tier"`
	Rank         string `bson:"rank" json:"rank"`
	LeaguePoints int    `bson:"leaguePoints" json:"leaguePoints"`
	Wins         int    `bson:"wins" json:"wins"`
	Losses       int    `bson:"losses" json:"losses"`
	
	// Metadata
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}