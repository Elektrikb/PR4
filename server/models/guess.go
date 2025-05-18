package models

import "time"

type GuessResponse struct {
	Message     string         `json:"message"`
	Attempt     *int           `json:"attempt,omitempty"`
	MaxAttempts *int           `json:"max_attempts,omitempty"`
	Black       *int           `json:"black,omitempty"`
	White       *int           `json:"white,omitempty"`
	IsEnd       *bool          `json:"is_end,omitempty"`
	TimeLeft    *time.Duration `json:"time_left,omitempty"`
}
