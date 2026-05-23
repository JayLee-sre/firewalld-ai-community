package model

import "time"

type Rule struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Severity       string    `json:"severity"`
	Enabled        bool      `json:"enabled"`
	Patterns       []string  `json:"patterns"`
	MatchLocations []string  `json:"match_locations"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type IPEntry struct {
	ID        string    `json:"id"`
	IPAddress string    `json:"ip_address"`
	ListType  string    `json:"list_type"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}
