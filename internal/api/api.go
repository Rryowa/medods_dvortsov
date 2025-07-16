package api

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/controller"
	"github.com/rryowa/medods_dvortsov/internal/storage"
	"github.com/rryowa/medods_dvortsov/internal/util"
)

const (
	shutdownTimeout = 5 * time.Second
)

type API struct {
	server           *echo.Echo
	controller       *controller.Controller
	log              *zap.SugaredLogger
	gracefulTimeout  time.Duration
	apiKeyRepository storage.APIKeyRepository
}

func NewAPI(c *controller.Controller, l *zap.SugaredLogger, sc *util.ServerConfig, apiKeyRepository storage.APIKeyRepository) *API {
	e := echo.New()

	e.Server.Addr = sc.ServerAddr
	e.Server.WriteTimeout = sc.WriteTimeout
	e.Server.ReadTimeout = sc.ReadTimeout
	e.Server.IdleTimeout = sc.IdleTimeout
	e.HTTPErrorHandler = ErrorHandler(l)

	return &API{
		server:           e,
		controller:       c,
		log:              l,
		gracefulTimeout:  sc.GracefulTimeout,
		apiKeyRepository: apiKeyRepository,
	}
}

func (a *API) Run(ctxBackground context.Context) {
	ctx, stop := signal.NotifyContext(ctxBackground, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	swagger, err := controller.GetSwagger()
	if err != nil {
		a.log.Fatalf("Failed to load OpenAPI specification: %v", err)
	}
	swagger.Servers = nil

	a.server.Use(APIKeyAuthMiddleware(a.apiKeyRepository))
	a.server.Use(echomiddleware.RequestLoggerWithConfig(GetLoggerMiddlewareConfig(a)))

	g := a.server.Group("/api")
	g.Use(middleware.OapiRequestValidator(swagger))
	/* Сгенерированный код сетапит маршруты OpenAPI и
	передает обработку запросов в методы из controller.gen.go.
	ServerInterfaceWrapper оборачивает методы контроллера и
	вызывает их, когда приходит соответствующий запрос.
	*/
	controller.RegisterHandlersWithBaseURL(a.server, a.controller, "/api")

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
