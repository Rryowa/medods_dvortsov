package api

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"go.uber.org/zap"

	"github.com/rryowa/medods_dvortsov/internal/controller"
)

const (
	shutdownTimeout = 5 * time.Second
)

type API struct {
	server          *echo.Echo
	controller      *controller.Controller
	zapLogger       *zap.SugaredLogger
	gracefulTimeout time.Duration
}

func NewAPI(c *controller.Controller, l *zap.SugaredLogger, sc *config.ServerConfig) *API {
	e := echo.New()

	e.Server.Addr = sc.ServerAddr
	e.Server.WriteTimeout = sc.WriteTimeout
	e.Server.ReadTimeout = sc.ReadTimeout
	e.Server.IdleTimeout = sc.IdleTimeout

	return &API{
		server:          e,
		controller:      c,
		zapLogger:       l,
		gracefulTimeout: sc.GracefulTimeout,
	}
}

func (a *API) Run(ctxBackground context.Context) {
	ctx, stop := signal.NotifyContext(ctxBackground, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	swagger, err := controller.GetSwagger()
	if err != nil {
		a.zapLogger.Fatalf("Failed to load OpenAPI specification: %v", err)
	}
	swagger.Servers = nil

	a.server.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogError:  true,

		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
			if v.Error == nil {
				a.zapLogger.Info("Request:",
					zap.String("Method", c.Request().Method),
					zap.String("URI", v.URI),
					zap.Int("Status", v.Status),
				)
			} else {
				a.zapLogger.Error("Request error",
					zap.String("Method", c.Request().Method),
					zap.String("URI", v.URI),
					zap.Int("Status", v.Status),
					zap.Error(v.Error),
				)
			}
			return nil
		},
	}))
	g := a.server.Group("/api")
	g.Use(middleware.OapiRequestValidator(swagger))
	// the generated code sets up the routing to match the OpenAPI spec and
	// delegates request handling to generated controller.gen.go methods.
	// The generated ServerInterfaceWrapper wraps tender.go methods and
	// calls tender.go methods when the corresponding route is accessed.
	controller.RegisterHandlersWithBaseURL(a.server, a.controller, "/api")

	a.ListenGracefulShutdown(ctx)
}

func (a *API) ListenGracefulShutdown(ctx context.Context) {
	go func() {
		if err := a.server.Start(a.server.Server.Addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.zapLogger.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()
	a.zapLogger.Infof("Listening on: %v\n", a.server.Server.Addr)

	<-ctx.Done()
	a.zapLogger.Info("Shutting down server...\n")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.zapLogger.Errorf("shutdown: %v", err)
	}

	longShutdown := make(chan struct{}, 1)

	go func() {
		time.Sleep(a.gracefulTimeout)
		longShutdown <- struct{}{}
	}()

	select {
	case <-shutdownCtx.Done():
		a.zapLogger.Errorf("server shutdown: %v", ctx.Err())
	case <-longShutdown:
		a.zapLogger.Infof("finished")
	}
}
