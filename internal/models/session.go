package models

import "time"

type RefreshSession struct {
	ID        int
	UserID    string
	TokenHash string
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
	CreatedAt time.Time
}
