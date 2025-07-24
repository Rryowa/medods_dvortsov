package api

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/controller"
	"github.com/rryowa/medods_dvortsov/internal/service"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

const (
	shutdownTimeout = 5 * time.Second
)

type API struct {
	server          *echo.Echo
	controller      *controller.Controller
	authService     *service.AuthService
	apiKeyService   *service.APIKeyService
	rdb             *redis.Client
	log             *zap.SugaredLogger
	gracefulTimeout time.Duration
	shutdownFuncs   []func()
}

func NewAPI(c *controller.Controller, authService *service.AuthService, rdb *redis.Client, aks *service.APIKeyService, sc *util.ServerConfig, l *zap.SugaredLogger, shutdownFuncs []func()) *API {
	e := echo.New()

	e.Server.Addr = sc.ServerAddr
	e.Server.WriteTimeout = sc.WriteTimeout
	e.Server.ReadTimeout = sc.ReadTimeout
	e.Server.IdleTimeout = sc.IdleTimeout
	e.HTTPErrorHandler = ErrorHandler(l)

	return &API{
		server:          e,
		controller:      c,
		authService:     authService,
		log:             l,
		gracefulTimeout: sc.GracefulTimeout,
		rdb:             rdb,
		apiKeyService:   aks,
		shutdownFuncs:   shutdownFuncs,
	}
}

func (a *API) Run(ctxBackground context.Context) {
	ctx, stop := signal.NotifyContext(ctxBackground, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	swagger, err := controller.GetSwagger()
	if err != nil {
		a.log.Fatalf("Failed to load OpenAPI specification: %v", err)
	}

	rateLimiterConfig := util.NewRateLimiterConfig()

	a.server.Use(echomiddleware.RequestLoggerWithConfig(LoggerMiddlewareConfig(a)))
	a.server.Use(RateLimiter(a.rdb, a.log, rateLimiterConfig))

	/*
		Сгенерированный код сетапит маршруты OpenAPI и
		передает обработку запросов в методы из controller.gen.go.
		ServerInterfaceWrapper оборачивает методы контроллера и
		вызывает их, когда приходит соответствующий запрос.
	*/
	openAPIWrapper := controller.ServerInterfaceWrapper{Handler: a.controller}

	// handle API key OR bearer token
	authenticator := NewAuthenticator(a.authService, a.apiKeyService)

	// OpenAPI request validator
	validatorOptions := &middleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: authenticator,
		},
	}
	validator := middleware.OapiRequestValidatorWithOptions(swagger, validatorOptions)

	v1 := a.server.Group("/api/v1")
	v1.Use(validator)

	controller.RegisterHandlers(v1, openAPIWrapper.Handler)

	a.ListenGracefulShutdown(ctx)
}

func (a *API) ListenGracefulShutdown(ctx context.Context) {
	go func() {
		err := a.server.Start(a.server.Server.Addr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()
	a.log.Infof("Listening on: %s", a.server.Server.Addr)
	a.log.Infof("random uuid: %s", uuid.New().String())

	<-ctx.Done()
	a.log.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := a.server.Shutdown(shutdownCtx)
	if err != nil {
		a.log.Errorf("shutdown: %v", err)
	}

	// После остановки сервера, офаем БД и Redis
	// ыql.DB.Close() ждет пока запросы обработаются
	for _, f := range a.shutdownFuncs {
		f()
	}

	longShutdown := make(chan struct{}, 1)

	go func() {
		time.Sleep(a.gracefulTimeout)
		longShutdown <- struct{}{}
	}()

	select {
	case <-shutdownCtx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			a.log.Info("server shutdown completed")
		} else {
			a.log.Errorf("server shutdown: %v", ctx.Err())
		}
	case <-longShutdown:
		a.log.Infof("finished")
	}
}
