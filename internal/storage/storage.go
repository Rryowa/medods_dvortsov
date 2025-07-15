package storage

import "context"

type Storage interface {
	Authentificator
}

type Authentificator interface {
	GetAccessAndRefreshTokens(ctx context.Context, username string) (string, string, error)
	UpdateAccessAndRefreshTokens(ctx context.Context, username, accessToken, refreshToken string) error
	GetGuidOfUser(ctx context.Context, username string) (string, error)
	UnathorizeUser(ctx context.Context, username string) error
}
