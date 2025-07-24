package models

import "time"

//nolint:gosec //file not handles sensitive data
const (
	MwSchemeAPIKeyAuth = "ApiKeyAuth"
	MwSchemeBearerAuth = "BearerAuth"

	MwAPIKeyHeader = "X-API-Key"

	MwUserIDKey = "userID"
	MwTokenKey  = "token"
)

type RefreshSession struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Selector       string    `json:"selector"`
	VerifierHash   string    `json:"verifier_hash"`
	UserAgent      string    `json:"user_agent"`
	IPAddress      string    `json:"ip_address"`
	AccessTokenJTI string    `json:"access_token_jti"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type RefreshToken struct {
	Token     string    `json:"token"`
	Hash      string    `json:"hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserMetadata struct {
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

type User struct {
	ID   int64  `json:"id"`
	GUID string `json:"guid"`
}
