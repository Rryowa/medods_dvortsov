//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=openapi/cfg.yaml openapi/yaml

package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

type Controller struct {
	zapLogger   *zap.SugaredLogger
	authService *service.AuthService
}

func NewController(logger *zap.SugaredLogger, authService *service.AuthService) *Controller {
	return &Controller{
		zapLogger:   logger,
		authService: authService,
	}
}

// GET /api/ping
func (c *Controller) CheckServer(ctx echo.Context) error {
	ctx.JSON(http.StatusOK, "ok")
	return nil
}

// POST /api/auth/token/issue
func (c *Controller) IssueTokens(ctx echo.Context) error {
	var req models.TokenIssueRequest
	if err := ctx.Bind(&req); err != nil {
		return InternalError(ctx, err)
	}

	ua := ctx.Request().UserAgent()
	ip := ctx.RealIP()

	access, refresh, err := c.authService.IssueTokens(ctx.Request().Context(), req.UserID, ua, ip)
	if err != nil {
		return InternalError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, models.TokenPairResponse{AccessToken: access, RefreshToken: refresh})
}

// POST /api/auth/token/refresh
func (c *Controller) RefreshTokens(ctx echo.Context) error {
	var req models.TokenRefreshRequest
	if err := ctx.Bind(&req); err != nil {
		return InternalError(ctx, err)
	}

	ua := ctx.Request().UserAgent() // FIXME: возможно нужно определять через библиотеку
	ip := ctx.RealIP()              // FIXME: возможно нужно определять через библиотеку

	access, refresh, err := c.authService.RefreshTokens(ctx.Request().Context(), req.AccessToken, req.RefreshToken, ua, ip)
	if err != nil {
		return InternalError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, models.TokenPairResponse{AccessToken: access, RefreshToken: refresh})
}

// POST /api/auth/logout
func (c *Controller) Logout(ctx echo.Context) error {
	var req models.LogoutRequest
	if err := ctx.Bind(&req); err != nil {
		return InternalError(ctx, err)
	}

	if err := c.authService.Logout(ctx.Request().Context(), req.UserID); err != nil {
		return InternalError(ctx, err)
	}
	return ctx.NoContent(http.StatusOK)
}

// GET /api/auth/user/guid (protected)
func (c *Controller) GetUserGUID(ctx echo.Context) error {
	token := extractBearer(ctx.Request().Header.Get("Authorization"))
	if token == "" {
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Reason: "missing bearer token"})
	}

	uid, err := c.authService.ValidateAccessToken(token)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, ErrorResponse{Reason: err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"user_id": uid})
}

func extractBearer(h string) string {
	const prefix = "Bearer "
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func InternalError(ctx echo.Context, err error) error {
	var customErr util.MyResponseError
	if errors.As(err, &customErr) {
		ctx.JSON(customErr.Status, ErrorResponse{Reason: customErr.Msg})
		return err
	}

	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Reason: err.Error()})
	return err
}
