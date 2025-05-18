package models

import "time"

type JoinResponse struct {
	Message   string        `json:"message"`
	Attempts  int           `json:"attempts"`
	TimeLimit time.Duration `json:"time_limit"`
	PlayerID  int           `json:"player_id"`
	GameID    int           `json:"game_id"`
}
