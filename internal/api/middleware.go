package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/storage"
)

const (
	APIKeyHeader       = "X-API-Key"
	ClientIDContextKey = "client_id"
)

// APIKeyAuthMiddleware проверяет наличие и валидность API ключа в заголовке X-API-Key.
// Если ключ валиден, client_id, ассоциированный с этим ключом, сохраняется в контексте Echo.
func APIKeyAuthMiddleware(apiKeyRepo storage.APIKeyRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get(APIKeyHeader)

			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "API key is missing")
			}

			// Look up the API key in the repository
			foundAPIKey, err := apiKeyRepo.GetAPIKey(c.Request().Context(), apiKey)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Error validating API key")
			}
			if foundAPIKey == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			c.Set(ClientIDContextKey, foundAPIKey.ClientID)

			return next(c)
		}
	}
}

func GetLoggerMiddlewareConfig(a *API) echomiddleware.RequestLoggerConfig {
	return echomiddleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogError:  true,

		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
			fields := []interface{}{
				"method", c.Request().Method,
				"uri", v.URI,
				"status", v.Status,
			}
			if v.Error != nil {
				fields = append(fields, zap.Error(v.Error))
			}
			if v.Error != nil {
				a.log.Errorw("Request", fields...)
			} else {
				a.log.Infow("Request", fields...)
			}
			return nil
		},
	}
}
