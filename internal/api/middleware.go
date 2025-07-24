package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/models"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

func NewAuthenticator(
	authService *service.AuthService,
	apiKeyService *service.APIKeyService,
) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		echoCtx, ok := ctx.Value(middleware.EchoContextKey).(echo.Context)
		if !ok {
			return errors.New("failed to get echo.Context from request context")
		}

		switch input.SecuritySchemeName {
		case models.MwSchemeAPIKeyAuth:
			apiKey := echoCtx.Request().Header.Get(models.MwAPIKeyHeader)
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "API key is missing")
			}
			valid, err := apiKeyService.IsValidAPIKey(ctx, apiKey)
			if err != nil {
				return fmt.Errorf("failed during api key validation: %w", err)
			}
			if !valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
			}
			return nil

		case models.MwSchemeBearerAuth:
			// Для /refresh access токен может быть просрочен
			if strings.HasSuffix(echoCtx.Request().URL.Path, "/auth/tokens/refresh") {
				return nil
			}

			authHeader := echoCtx.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is missing")
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header format must be Bearer {token}")
			}

			token := strings.TrimPrefix(authHeader, bearerPrefix)
			echoCtx.Set(models.MwTokenKey, token)

			userID, err := authService.AuthenticateAccessToken(ctx, token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			echoCtx.Set(models.MwUserIDKey, userID)
			return nil

		default:
			return fmt.Errorf("unknown security scheme: %s", input.SecuritySchemeName)
		}
	}
}

// RateLimiter middleware
//
//	interval: окно для подсчета запросов
//	blockTime: Длительность блокировки после превышения лимита (0 = без блокировки)
func RateLimiter(
	redisClient *redis.Client,
	_ *zap.SugaredLogger,
	config *util.RateLimiterConfig,
) echo.MiddlewareFunc {
	/*
		Без Lua скрипта:
		Между INCR и ExpireNX могут вклиниться другие запросы
		Race confidtion при увеличении счетчика
	*/
	// Атомарное выполнение
	script := redis.NewScript(`
        local count_key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local interval = tonumber(ARGV[2])
		local block_time = tonumber(ARGV[3])

		local current = redis.call("GET", count_key)
		current = current and tonumber(current) or 0

		if current >= limit then
			local block_key = count_key .. ":block"
			local blocked = redis.call("GET", block_key)
			
			if not blocked then
				-- Блокируемся
				redis.call("SETEX", block_key, block_time, "1")
			end
			
			-- Превышен лимит
			return 0
		end

		redis.call("INCR", count_key)

		-- TTL при первом запросе
		if current == 0 then
			redis.call("EXPIRE", count_key, interval)
		end

		-- Ок
		return 1
    `)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ipAddress := c.RealIP()
			key := "rate_limit:" + ipAddress

			// Атомарно
			result, err := script.Run(
				c.Request().Context(),
				redisClient,
				[]string{key},
				config.Limit,
				int(config.Interval.Seconds()),
				int(config.BlockTime.Seconds()),
			).Int()
			if err != nil {
				c.Logger().Errorf("Rate limiter error: %v", err)
				return next(c) // Пропускаем запрос
			}

			if result == 0 {
				// Превышение лимита
				c.Response().Header().Set("Retry-After", strconv.Itoa(int(config.BlockTime.Seconds())))
				return echo.NewHTTPError(http.StatusTooManyRequests, "Too many requests")
			}

			return next(c)
		}
	}
}

func LoggerMiddlewareConfig(a *API) echomiddleware.RequestLoggerConfig {
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
				fields = append(fields, "error", fmt.Sprintf("%+v", v.Error))
				a.log.Errorw("Request", fields...)
			} else {
				a.log.Infow("Request", fields...)
			}
			return nil
		},
	}
}
