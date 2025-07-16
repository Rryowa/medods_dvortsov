package models

import "time"

type RefreshSession struct {
	ID           string    `json:"id"` // Stores the raw, base64 encoded refresh token
	UserID       string    `json:"user_id"`
	UserAgent    string    `json:"user_agent"`
	ClientID     string    `json:"client_id"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type RefreshToken struct {
	Token     string    `json:"token"`
	Hash      string    `json:"hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserMetadata struct {
	UserAgent string `json:"user_agent"`
	ClientID  string `json:"client_id"`
	IPAddress string `json:"ip_address"`
}

type APIKey struct {
	Key      string `json:"key"`
	ClientID string `json:"client_id"`
}
