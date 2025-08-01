package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/storage"
)

func ErrorHandler(log *zap.SugaredLogger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		if isUnauthorizedTokenError(err) {
			c.JSON(http.StatusUnauthorized, map[string]string{"reason": err.Error()})
			return
		}

		var he *echo.HTTPError
		if errors.As(err, &he) {
			if he.Code == http.StatusInternalServerError {
				log.Errorw("HTTP error", "error", err, "uri", c.Request().RequestURI)
			}
			//nolint:errcheck // useless here
			if err := c.JSON(he.Code, map[string]string{"reason": he.Message.(string)}); err != nil {
				log.Errorw("failed to write json response", "error", err)
			}
			return
		}

		log.Errorw("unhandled error", "error", err, "uri", c.Request().RequestURI)
		c.JSON(http.StatusInternalServerError, map[string]string{"reason": "internal server error"})
	}
}

func isUnauthorizedTokenError(err error) bool {
	return errors.Is(err, service.ErrTokenExpired) ||
		errors.Is(err, service.ErrTokenInvalid) ||
		errors.Is(err, storage.ErrSessionNotFound)
}
