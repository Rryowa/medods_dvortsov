package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rryowa/medods_dvortsov/internal/storage"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

var (
	ErrTokenExpired         = errors.New("token expired")
	ErrTokenInvalid         = errors.New("token invalid")
	ErrTokenMalformed       = errors.New("token is malformed")
	ErrTokenRevoked         = errors.New("token revoked")
	ErrInvalidUserID        = errors.New("invalid userID")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)

type TokenService struct {
	JwtSecretKey []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	tokenStorage storage.TokenStorage
}

func NewTokenService(cfg *util.TokenConfig, tokenStorage storage.TokenStorage) *TokenService {
	return &TokenService{
		JwtSecretKey: cfg.JwtSecretKey,
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
		tokenStorage: tokenStorage,
	}
}

type jwtClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// CreateAccessToken создает SHA512 signed access токен с новым JTI
func (ts *TokenService) CreateAccessToken(userID int64, now time.Time) (string, string, error) {
	jti := uuid.NewString()
	signedToken, err := ts.CreateAccessTokenWithJTI(userID, now, jti)
	if err != nil {
		return "", "", err
	}
	return signedToken, jti, nil
}

// CreateAccessTokenWithJTI создает SHA512 signed access токен с JTI
func (ts *TokenService) CreateAccessTokenWithJTI(userID int64, now time.Time, jti string) (string, error) {
	claims := &jwtClaims{
		UserID: strconv.FormatInt(userID, 10),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signedToken, err := token.SignedString(ts.JwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("signed string: %w", err)
	}

	return signedToken, nil
}

func (ts *TokenService) CreateRefreshToken() (token, selector, verifierHash string, err error) {
	rawToken := make([]byte, util.RawTokenLength)
	if _, err = rand.Read(rawToken); err != nil {
		return "", "", "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	selector = base64.RawURLEncoding.EncodeToString(rawToken[:16])
	verifier := base64.RawURLEncoding.EncodeToString(rawToken[16:])

	hashedVerifierBytes := sha256.Sum256([]byte(verifier))
	verifierHash = hex.EncodeToString(hashedVerifierBytes[:])

	token = selector + "." + verifier

	return token, selector, verifierHash, nil
}

func (ts *TokenService) ValidateRefreshToken(token, verifierHash string) error {
	parts := strings.Split(token, ".")
	if len(parts) != util.TokenPartsExpected {
		return errors.New("invalid token format")
	}

	verifier := parts[1]

	hashedVerifierBytes, err := hex.DecodeString(verifierHash)
	if err != nil {
		return fmt.Errorf("failed to decode stored hash: %w", err)
	}

	newHashBytes := sha256.Sum256([]byte(verifier))

	if subtle.ConstantTimeCompare(newHashBytes[:], hashedVerifierBytes) != 1 {
		return errors.New("invalid refresh token")
	}

	return nil
}

func (ts *TokenService) ValidateAccessTokenAndGetUserID(ctx context.Context, token string) (int64, error) {
	isInvalidated, err := ts.IsAccessTokenInvalidated(ctx, token)
	if err != nil {
		return 0, fmt.Errorf("failed to check if token is invalidated: %w", err)
	}
	if isInvalidated {
		return 0, ErrTokenRevoked
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithLeeway(util.JWTLeeWay),
		jwt.WithExpirationRequired(),
	}

	parsedToken, err := jwt.ParseWithClaims(
		token,
		&jwtClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
				return nil, ErrInvalidSigningMethod
			}
			return ts.JwtSecretKey, nil
		},
		opts...,
	)
	if err != nil {
		return 0, fmt.Errorf("parse token claims: %w", err)
	}

	if parsedToken == nil || !parsedToken.Valid {
		return 0, ErrTokenInvalid
	}

	claims, ok := parsedToken.Claims.(*jwtClaims)
	if !ok || claims.UserID == "" {
		return 0, ErrTokenInvalid
	}

	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		return 0, ErrInvalidUserID
	}

	return userID, nil
}

func (ts *TokenService) InvalidateAccessToken(ctx context.Context, accessToken string) error {
	claims, err := ts.getClaimsFromToken(accessToken)
	if err != nil {
		return fmt.Errorf("get claims from token: %w", err)
	}

	expiration := time.Until(claims.ExpiresAt.Time)

	if err := ts.tokenStorage.InvalidateToken(ctx, accessToken, expiration); err != nil {
		return fmt.Errorf("invalidate token: %w", err)
	}
	return nil
}

// IsAccessTokenInvalidated проверяет, находится ли токен в черном списке
// Это первый шаг валидации токена, до проверки подписи и срока действия
func (ts *TokenService) IsAccessTokenInvalidated(ctx context.Context, accessToken string) (bool, error) {
	isInvalidated, err := ts.tokenStorage.IsTokenInvalidated(ctx, accessToken)
	if err != nil {
		return false, fmt.Errorf("is token invalidated: %w", err)
	}
	return isInvalidated, nil
}

func (ts *TokenService) getClaimsFromToken(token string) (*jwtClaims, error) {
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token, &jwtClaims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenMalformed, err)
	}

	claims, ok := parsedToken.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
