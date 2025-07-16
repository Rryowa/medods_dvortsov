package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/service"
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


		he, ok := err.(*echo.HTTPError)
		if ok {
			if he.Code == http.StatusInternalServerError {
				log.Errorw("HTTP error", "error", err, "uri", c.Request().RequestURI)
			}
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
		errors.Is(err, service.ErrRefreshTokenNotFoundOrUsed)
}
