package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rryowa/medods_dvortsov/internal/util"
)

type TokenService struct {
	JwtSecretKey []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

func NewTokenService(cfg *util.TokenConfig) *TokenService {
	return &TokenService{
		JwtSecretKey: cfg.JwtSecretKey,
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
	}
}

type jwtClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired               = errors.New("token expired")
	ErrTokenInvalid               = errors.New("token invalid")
	ErrTokenRevoked               = errors.New("token revoked")
	ErrInvalidUserID              = errors.New("invalid userID")
	ErrInvalidSigningMethod       = errors.New("invalid signing method")
	ErrRefreshTokenNotFoundOrUsed = errors.New("refresh token not found or already used")
)

// CreateAccessToken создает SHA512 signed access токен
//
//   - Записывает bcrypt токен в бд
//   - Возвращает signed access токен в формате JWT
//
// Формат токена - JWT.
// Алгоритм для подписи токена - SHA512.
// Хранить токен в базе строго запрещено.
func (ts *TokenService) CreateAccessToken(userID string, now time.Time) (string, error) {
	claims := &jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString(ts.JwtSecretKey)
}

// CreateRefreshToken создает refresh токен
//
//	Возвращает:
//	  - refresh токен в формате base64
//	  - bcrypt хеш токена
//	  - ошибку
//
// Формат токена - произвольный.
// Передаваться токен должен только в формате `base64`.
// Хранить токен в базе строго в виде `bcrypt` хеша.
// Токен должен быть защищен от повторного использования.
// Токен должен быть защищен от изменений на стороне клиента.
func (ts *TokenService) CreateRefreshToken(userID string, now time.Time) (string, error) {
	tokenBytes := make([]byte, 32) // 256 бит
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	// base64 без паддинга т.к. `=` может быть некорректно обработан
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	return token, nil
}

// ValidateRefreshToken проверяет refresh токен
//
//   - Декодирует токен из base64
//   - Сравнивает с хэшем в базе
//   - Возвращает true, если токен валидный, иначе false
// func (ts *TokenService) ValidateRefreshToken(token string) error {
// 	storedHash, expiresAt, err := ts.sessionManager.GetRefreshToken(receivedToken)
// 	if err != nil {
// 		return fmt.Errorf("failed to get refresh token: %w", err)
// 	}
//
// 	if expiresAt.Before(time.Now()) {
// 		return ErrTokenExpired
// 	}
//
// 	receivedToken, err := base64.RawURLEncoding.DecodeString(receivedToken)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode refresh token: %w", err)
// 	}
//
// 	err = bcrypt.CompareHashAndPassword(storedHash, receivedToken)
// 	if err != nil {
// 		return ErrTokenInvalid
// 	}
//
// 	return nil
// }

// ValidateAndGetUserID - для строгой валидации (аутентификация)
func (ts *TokenService) ValidateAccessTokenAndGetUserID(token string) (string, error) {
	return ts.getUserID(token, true)
}

// ValidateAndGetUserIDFromExpired - не строгая валидация (обновление токена)
//
//	Когда access token истекает, клиент отправляет refresh token для получения новой пары
//	В таком случае нам надо извлечь userID из expired токена
func (ts *TokenService) ValidateAndGetUserIDFromExpired(token string) (string, error) {
	return ts.getUserID(token, false)
}

func (ts *TokenService) getUserID(token string, strict bool) (string, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}),
		jwt.WithLeeway(5 * time.Second),
	}

	if strict {
		opts = append(opts, jwt.WithExpirationRequired())
	} else {
		opts = append(opts, jwt.WithoutClaimsValidation())
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
		return "", err
	}

	if parsedToken == nil || (strict && !parsedToken.Valid) {
		return "", ErrTokenInvalid
	}

	claims, ok := parsedToken.Claims.(*jwtClaims)
	if !ok || claims.UserID == "" {
		return "", ErrTokenInvalid
	}

	return claims.UserID, nil
}

// TODO: Implement
// func (ts *TokenService) RevokeToken(jti string, expiry time.Duration) error {
// 	ctx := context.Background()

// 	// ключ с TTL = времени жизни токена
// 	// (автоматически удалится после истечения срока)
// 	return rdb.Set(ctx, "revoked:"+jti, "1", expiry).Err()
// }
