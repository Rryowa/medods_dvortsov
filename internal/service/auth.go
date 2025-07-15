package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rryowa/medods_dvortsov/internal/models"
)

type AuthService struct {
	refreshManager RefreshTokenManager
	secretKey      []byte
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

func NewAuthService(m RefreshTokenManager, secretKey []byte) *AuthService {
	return &AuthService{
		refreshManager: m,
		secretKey:      secretKey,
		accessTTL:      15 * time.Minute,
		refreshTTL:     24 * time.Hour,
	}
}

func (s *AuthService) IssueTokens(ctx context.Context, userID, ua, ip string) (access, refresh string, err error) {
	now := time.Now().UTC()

	access, err = s.signAccessToken(userID, now)
	if err != nil {
		return "", "", err
	}

	refresh, err = generateRandomToken()
	if err != nil {
		return "", "", err
	}

	session := models.RefreshSession{
		RefreshToken: refresh,
		UserID:       userID,
		UserAgent:    ua,
		IPAddress:    ip,
		ExpiresAt:    now.Add(s.refreshTTL),
		CreatedAt:    now,
	}

	if err = s.refreshManager.CreateSession(ctx, session, s.refreshTTL); err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *AuthService) RefreshTokens(
	ctx context.Context,
	expiredAccess, refresh, ua, ip string,
) (newAccess, newRefresh string, err error) {
	userID, err := s.extractUserID(expiredAccess)
	if err != nil {
		return "", "", err
	}

	session, err := s.refreshManager.FindSession(ctx, refresh)
	if err != nil {
		return "", "", err
	}
	if session == nil {
		return "", "", errors.New("refresh session not found") // TODO: custom error type
	}

	if session.UserID != userID {
		return "", "", errors.New("token pair mismatch")
	}

	if session.UserAgent != ua {
		_ = s.refreshManager.DeleteAllUserSessions(ctx, userID) // best-effort
		return "", "", errors.New("user-agent changed; all sessions revoked")
	}

	_ = s.refreshManager.DeleteSession(ctx, refresh) // best-effort

	return s.IssueTokens(ctx, userID, ua, ip)
}

func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return s.refreshManager.DeleteAllUserSessions(ctx, userID)
}

// FIXME: move to util
type jwtClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func (s *AuthService) signAccessToken(userID string, now time.Time) (string, error) {
	claims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(s.secretKey)
}

func (s *AuthService) extractUserID(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&jwtClaims{},
		func(t *jwt.Token) (interface{}, error) { return s.secretKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithoutClaimsValidation(), // ignore expiration
	)
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return "", err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || claims.UserID == "" {
		return "", errors.New("invalid access token")
	}
	return claims.UserID, nil
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&jwtClaims{},
		func(t *jwt.Token) (interface{}, error) { return s.secretKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
	)
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid access token")
	}
	return claims.UserID, nil
}
