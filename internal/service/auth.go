package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type AuthService struct {
	tokenService      TokenService
	sessionRepository storage.SessionRepository
	log               *zap.SugaredLogger
}

func NewAuthService(ts TokenService, sr storage.SessionRepository, log *zap.SugaredLogger) *AuthService {
	return &AuthService{
		tokenService:      ts,
		sessionRepository: sr,
		log:               log,
	}
}

func (as *AuthService) IssueTokens(
	ctx context.Context,
	userID string,
	userMetadata models.UserMetadata,
) (accessToken, refreshToken string, err error) {
	now := time.Now().UTC()

	accessToken, err = as.tokenService.CreateAccessToken(userID, now)
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token: %w", err)
	}

	refreshToken, err = as.tokenService.CreateRefreshToken(userID, now)
	if err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	session := models.RefreshSession{
		ID:        refreshToken, // Use the raw, base64 encoded token as the ID
		UserID:    userID,
		UserAgent: userMetadata.UserAgent,
		ClientID:  userMetadata.ClientID,
		IPAddress: userMetadata.IPAddress,
		CreatedAt: now,
		ExpiresAt: now.Add(as.tokenService.refreshTTL),
	}

	as.log.Debugw("Creating new session", "sessionID", session.ID, "userID", session.UserID)
	if err = as.sessionRepository.CreateSession(ctx, session, as.tokenService.refreshTTL); err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}

	return accessToken, refreshToken, nil
}

// TODO: 1. Найти хеш в базе по связанному user_id
// 2. Проверить срок действия
// 3. Если VerifyRefreshToken пройден: - ПОМЕТИТЬ токен как использованный (UPDATE tokens SET used = true)
// 4. УДАЛИТЬ/АННУЛИРОВАТЬ все токены пользователя при подозрении
//
// TODO: Защита refresh-токена:
// - Хранить в HttpOnly, Secure, SameSite=Strict cookie
// - Требовать re-authentication для критичных операций
// - Валидировать fingerprint устройства

func (as *AuthService) RefreshTokens(
	ctx context.Context,
	refreshToken string,
	userMetadata models.UserMetadata,
) (newAccessToken, newRefreshToken string, err error) {
	session, err := as.sessionRepository.GetCurrentSession(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return "", "", ErrRefreshTokenNotFoundOrUsed
		}
		return "", "", fmt.Errorf("failed to find session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
		return "", "", ErrTokenExpired
	}

	if session.UserAgent != userMetadata.UserAgent {
		if err = as.sessionRepository.DeleteAllUserSessions(ctx, session.UserID); err != nil {
			return "", "", fmt.Errorf("failed to delete all user sessions: %w", err)
		}
		return "", "", errors.New("user-agent changed; all sessions revoked")
	}

	if err = as.sessionRepository.DeleteSession(ctx, refreshToken); err != nil {
		return "", "", fmt.Errorf("failed to delete session: %w", err)
	}

	return as.IssueTokens(ctx, session.UserID, userMetadata)
}

func (as *AuthService) Logout(ctx context.Context, userID string) error {
	if err := as.sessionRepository.DeleteAllUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete all user sessions: %w", err)
	}
	return nil
}
