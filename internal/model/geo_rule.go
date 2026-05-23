package model

import "time"

type GeoRule struct {
	ID          string    `json:"id"`
	Country     string    `json:"country"`
	CountryCode string    `json:"country_code"`
	Action      string    `json:"action"` // "block" or "allow"
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}
