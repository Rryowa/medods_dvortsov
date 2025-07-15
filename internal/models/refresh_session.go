package models

import "time"

// RefreshSession represents an active refresh-token session for a user.
type RefreshSession struct {
	RefreshToken string    `json:"refresh_token"`
	UserID       string    `json:"user_id"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}
