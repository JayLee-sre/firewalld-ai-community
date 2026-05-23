package model

import "time"

// User represents a dashboard user account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin", "operator", "viewer"
	CreatedAt    time.Time `json:"created_at"`
}
