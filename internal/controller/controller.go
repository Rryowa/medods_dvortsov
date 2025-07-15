//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=openapi/cfg.yaml openapi/yaml

package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

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

// (GET /api/ping).
func (c *Controller) CheckServer(ctx echo.Context) error {
	ctx.JSON(http.StatusOK, "ok")
	return nil
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
