package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/storage"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

type AuthService struct {
	tokenService   *TokenService
	storage        storage.Storage
	webhookService *WebhookService
	log            *zap.SugaredLogger
}

func NewAuthService(ts *TokenService, s storage.Storage, ws *WebhookService, log *zap.SugaredLogger) *AuthService {
	return &AuthService{
		tokenService:   ts,
		storage:        s,
		webhookService: ws,
		log:            log,
	}
}

func (as *AuthService) AuthenticateAccessToken(ctx context.Context, tokenString string) (int64, error) {
	userID, err := as.tokenService.ValidateAccessTokenAndGetUserID(ctx, tokenString)
	if err != nil {
		return 0, fmt.Errorf("access token validation failed: %w", err)
	}
	as.log.Debugw("successfully authenticated access token", "userID", userID)
	return userID, nil
}

// IssueTokens выпускает новую пару токенов
// userMetadata (IP, User-Agent) используется для привязки сессии к клиенту
func (as *AuthService) IssueTokens(
	ctx context.Context,
	guid string,
	userMetadata models.UserMetadata,
) (accessToken, refreshToken string, err error) {
	as.log.Debugw("issuing tokens", "guid", guid)
	now := time.Now().UTC()

	jti := uuid.NewString()

	refreshToken, selector, verifierHash, err := as.tokenService.CreateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	session := models.RefreshSession{
		Selector:       selector,
		VerifierHash:   verifierHash,
		UserAgent:      userMetadata.UserAgent,
		IPAddress:      userMetadata.IPAddress,
		AccessTokenJTI: jti,
		CreatedAt:      now,
		ExpiresAt:      now.Add(as.tokenService.refreshTTL),
	}

	user, err := as.storage.IssueTokensTx(ctx, guid, session)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute issue tokens transaction: %w", err)
	}

	accessToken, err = as.tokenService.CreateAccessTokenWithJTI(user.ID, now, jti)
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token with correct user ID: %w", err)
	}

	as.log.Debugw("successfully issued tokens", "userID", user.ID, "guid", user.GUID)

	return accessToken, refreshToken, nil
}

func (as *AuthService) detectTheftAndRevoke(ctx context.Context, selector string) error {
	usedSession, err := as.storage.FindSessionBySelector(ctx, selector)
	if err != nil {
		// Сессия не найдена - невалиден, а не украден
		return storage.ErrSessionNotFound
	}

	as.log.Warnw("token reuse detected, all sessions revoked", "selector", selector, "userID", usedSession.UserID)
	if err := as.storage.DeleteAllUserSessions(ctx, usedSession.UserID); err != nil {
		return fmt.Errorf("failed to revoke sessions after theft detection: %w", err)
	}

	return errors.New("token reuse detected, all sessions revoked")
}

// RefreshTokens обновляет пару токенов
// старый refresh-токен помечается как использованный - при попытке повторного
// использования отзываются все сессии пользователя.
func (as *AuthService) RefreshTokens(
	ctx context.Context,
	accessToken, refreshToken string,
	userMetadata models.UserMetadata,
) (newAccessToken, newRefreshToken string, err error) {
	// Парсим access, чтобы получить JTI
	claims, err := as.tokenService.getClaimsFromToken(accessToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid access token: %w", err)
	}

	// Найти активную сессию по refresh-токену.
	parts := strings.Split(refreshToken, ".")
	if len(parts) != util.TokenPartsExpected {
		return "", "", errors.New("invalid refresh token format")
	}
	selector := parts[0]
	activeSession, err := as.storage.GetActiveSessionBySelector(ctx, selector)
	if err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return "", "", as.detectTheftAndRevoke(ctx, selector)
		}
		return "", "", fmt.Errorf("failed to get active session: %w", err)
	}
	if activeSession.AccessTokenJTI != claims.ID {
		return "", "", errors.New("access and refresh tokens do not match")
	}

	// Проверка User-Agent
	if activeSession.UserAgent != userMetadata.UserAgent {
		as.log.Warnw("user-agent mismatch, revoking all sessions", "sessionID", activeSession.ID)
		if err := as.storage.DeleteAllUserSessions(ctx, activeSession.UserID); err != nil {
			return "", "", fmt.Errorf("failed to revoke sessions after user-agent mismatch: %w", err)
		}
		return "", "", errors.New("user-agent has changed, all sessions revoked")
	}

	// Проверка ip
	// comment if condition to test webhook
	if activeSession.IPAddress != userMetadata.IPAddress {
		as.log.Infow("ip address changed, sending webhook notification", "sessionID", activeSession.ID)
		as.webhookService.NotifyIPChange(ctx, map[string]any{
			"user_id":    activeSession.UserID,
			"old_ip":     activeSession.IPAddress,
			"new_ip":     userMetadata.IPAddress,
			"user_agent": userMetadata.UserAgent,
		})
	}

	if err := as.tokenService.ValidateRefreshToken(refreshToken, activeSession.VerifierHash); err != nil {
		return "", "", err
	}

	// Rotation
	now := time.Now().UTC()
	newAccessToken, newJTI, err := as.tokenService.CreateAccessToken(activeSession.UserID, now)
	if err != nil {
		return "", "", fmt.Errorf("failed to create new access token: %w", err)
	}

	newRefreshToken, newSelector, newVerifierHash, err := as.tokenService.CreateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to create new refresh token: %w", err)
	}

	newSession := models.RefreshSession{
		Selector:       newSelector,
		VerifierHash:   newVerifierHash,
		UserAgent:      userMetadata.UserAgent,
		IPAddress:      userMetadata.IPAddress,
		AccessTokenJTI: newJTI,
		CreatedAt:      now,
		ExpiresAt:      now.Add(as.tokenService.refreshTTL),
	}

	_, err = as.storage.RotateTokensTx(ctx, selector, newSession, activeSession.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute rotate tokens transaction: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout отзывает access-токен и удаляет все refresh-сессии пользователя.
func (as *AuthService) Logout(ctx context.Context, accessToken string) error {
	userID, err := as.tokenService.ValidateAccessTokenAndGetUserID(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("access token validation failed: %w", err)
	}

	if err := as.tokenService.InvalidateAccessToken(ctx, accessToken); err != nil {
		return fmt.Errorf("failed to invalidate access token: %w", err)
	}

	if err := as.storage.DeleteAllUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete all user sessions: %w", err)
	}

	return nil
}

func (as *AuthService) GetPublicGUID(ctx context.Context, userID int64) (string, error) {
	as.log.Debugw("getting public guid", "userID", userID)
	user, err := as.storage.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user by id: %w", err)
	}
	return user.GUID, nil
}
