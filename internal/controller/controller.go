//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=../openapi/cfg.yaml ../openapi/openapi.yaml

package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

type Controller struct {
	authService *service.AuthService
	log         *zap.SugaredLogger
}

func NewController(as *service.AuthService, l *zap.SugaredLogger) *Controller {
	return &Controller{
		authService: as,
		log:         l,
	}
}

// IssueTokens (POST /api/auth/tokens)
func (c *Controller) IssueTokens(ctx echo.Context, params IssueTokensParams) error {
	req := ctx.Request()
	userAgent := req.UserAgent()
	ipAddress := ctx.RealIP()

	access, refresh, err := c.authService.IssueTokens(
		req.Context(),
		params.Guid.String(),
		models.UserMetadata{
			UserAgent: userAgent,
			IPAddress: ipAddress,
		},
	)
	if err != nil {
		return err
	}

	setRefreshCookie(ctx, refresh)

	return ctx.JSON(http.StatusOK, TokensResponse{AccessToken: access})
}

// RefreshTokens (POST /api/auth/tokens/refresh)
func (c *Controller) RefreshTokens(ctx echo.Context) error {
	refreshCookie, err := ctx.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "refresh token not found")
	}
	refreshToken := refreshCookie.Value

	if refreshToken == "" || refreshToken == "null" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or empty refresh token from cookie")
	}

	authHeader := ctx.Request().Header.Get("Authorization")
	if authHeader == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is missing")
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header format must be Bearer {token}")
	}
	accessToken := strings.TrimPrefix(authHeader, bearerPrefix)

	req := ctx.Request()
	userAgent := req.UserAgent()
	ipAddress := ctx.RealIP()

	newAccess, newRefresh, err := c.authService.RefreshTokens(
		ctx.Request().Context(),
		accessToken,
		refreshToken,
		models.UserMetadata{
			UserAgent: userAgent,
			IPAddress: ipAddress,
		},
	)
	if err != nil {
		// session/token expiration
		if errors.Is(err, storage.ErrSessionNotFound) || errors.Is(err, service.ErrTokenExpired) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
		}

		// malformed token, JTI mismatch etc
		c.log.Errorw("failed to refresh tokens", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	setRefreshCookie(ctx, newRefresh)

	return ctx.JSON(http.StatusOK, TokensResponse{AccessToken: newAccess})
}

func (c *Controller) Logout(ctx echo.Context) error {
	token, ok := ctx.Get(models.MwTokenKey).(string)
	if !ok || token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "access token not found in context")
	}
	if err := c.authService.Logout(ctx.Request().Context(), token); err != nil {
		return err
	}

	clearRefreshCookie(ctx)

	return ctx.NoContent(http.StatusNoContent)
}

// GetUserGUID возвращает публичный GUID пользователя.
// Внутренний userID (int64) извлекается из контекста, куда он был добавлен
// middleware аутентификации после проверки access-токена.
func (c *Controller) GetUserGUID(ctx echo.Context) error {
	userID, ok := ctx.Get(models.MwUserIDKey).(int64)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "user ID not found in context")
	}

	guid, err := c.authService.GetPublicGUID(ctx.Request().Context(), userID)
	if err != nil {
		return fmt.Errorf("get public GUID: %w", err)
	}

	return ctx.JSON(http.StatusOK, UserGUIDResponse{UserId: uuid.MustParse(guid)})
}

func setRefreshCookie(ctx echo.Context, token string) {
	cookie := new(http.Cookie)
	cookie.Name = "refresh_token"
	cookie.Value = token
	cookie.HttpOnly = true
	cookie.Path = "/api/v1/auth"
	//FIXME: В prod должен быть true
	cookie.Secure = false
	cookie.SameSite = http.SameSiteStrictMode
	ctx.SetCookie(cookie)
}

func clearRefreshCookie(ctx echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = "refresh_token"
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.HttpOnly = true
	cookie.Path = "/api/v1/auth"
	ctx.SetCookie(cookie)
}
