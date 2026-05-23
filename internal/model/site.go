package model

import "time"

type Site struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Domains          []string  `json:"domains"`
	Upstream         string    `json:"upstream"`
	Enabled          bool      `json:"enabled"`
	AIEnabled        bool      `json:"ai_enabled"`
	ChallengeEnabled bool      `json:"challenge_enabled"`
	SiteType         string    `json:"site_type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
