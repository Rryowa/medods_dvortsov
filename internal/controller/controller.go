//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=../../openapi/cfg.yaml ../../openapi/openapi.yaml

package controller

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/service"
)

type Controller struct {
	zapLogger    *zap.SugaredLogger
	authService  *service.AuthService
	tokenService *service.TokenService
}

func NewController(logger *zap.SugaredLogger, authService *service.AuthService, tokenService *service.TokenService) *Controller {
	return &Controller{
		zapLogger:    logger,
		authService:  authService,
		tokenService: tokenService,
	}
}

// Healthcheck (GET /api/ping)
func (c *Controller) Healthcheck(ctx echo.Context) error {
	ctx.JSON(http.StatusOK, "ok")
	return nil
}

// IssueTokens (POST /api/auth/token/issue)
func (c *Controller) IssueTokens(ctx echo.Context, params IssueTokensParams) error {
	req := ctx.Request()
	userAgent := req.UserAgent()
	clientID := ctx.Get("client_id").(string)
	ipAddress := ctx.RealIP()

	access, refresh, err := c.authService.IssueTokens(
		req.Context(),
		params.UserId.String(),
		models.UserMetadata{
			UserAgent: userAgent,
			ClientID:  clientID,
			IPAddress: ipAddress,
		},
	)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, TokenPairResponse{AccessToken: access, RefreshToken: refresh})
}

// RefreshTokens (POST /api/auth/token/refresh)
func (c *Controller) RefreshTokens(ctx echo.Context) error {
	var tokenRefreshReq TokenRefreshRequest
	if err := ctx.Bind(&tokenRefreshReq); err != nil {
		return err
	}

	req := ctx.Request()
	userAgent := req.UserAgent()
	clientID := ctx.Get("client_id").(string)
	ipAddress := ctx.RealIP()

	access, refresh, err := c.authService.RefreshTokens(
		ctx.Request().Context(),
		tokenRefreshReq.RefreshToken,
		models.UserMetadata{
			UserAgent: userAgent,
			ClientID:  clientID,
			IPAddress: ipAddress,
		},
	)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, TokenPairResponse{AccessToken: access, RefreshToken: refresh})
}

// Logout (POST /api/auth/logout)
func (c *Controller) Logout(ctx echo.Context) error {
	var req LogoutRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}

	if err := c.authService.Logout(ctx.Request().Context(), req.UserId.String()); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// GetUserGUID (GET /api/auth/user/guid) protected
func (c *Controller) GetUserGUID(ctx echo.Context) error {
	token := extractBearer(ctx.Request().Header.Get("Authorization"))
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
	}

	uid, err := c.tokenService.ValidateAccessTokenAndGetUserID(token)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, UserGUIDResponse{UserId: uuid.MustParse(uid)})
}

func extractBearer(h string) string {
	const prefix = "Bearer "
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}
