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
	"github.com/rryowa/medods_dvortsov/internal/util"
)

const (
	shutdownTimeout = 5 * time.Second
)

type API struct {
	server          *echo.Echo
	controller      *controller.Controller
	log             *zap.SugaredLogger
	gracefulTimeout time.Duration
}

func NewAPI(c *controller.Controller, l *zap.SugaredLogger, sc *util.ServerConfig) *API {
	e := echo.New()

	e.Server.Addr = sc.ServerAddr
	e.Server.WriteTimeout = sc.WriteTimeout
	e.Server.ReadTimeout = sc.ReadTimeout
	e.Server.IdleTimeout = sc.IdleTimeout

	return &API{
		server:          e,
		controller:      c,
		log:             l,
		gracefulTimeout: sc.GracefulTimeout,
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

	a.server.Use(echomiddleware.RequestLoggerWithConfig(getMiddlewareConfig(a)))

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
	a.log.Infof("Listening on: %v", a.server.Server.Addr)

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

func getMiddlewareConfig(a *API) echomiddleware.RequestLoggerConfig {
	return echomiddleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogError:  true,

		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
			if v.Error != nil {
				a.log.Errorw("Request error",
					"method", c.Request().Method,
					"uri", v.URI,
					"status", v.Status,
					"error", v.Error,
				)
			} else {
				a.log.Infow("Request",
					"method", c.Request().Method,
					"uri", v.URI,
					"status", v.Status,
				)
			}

			return nil
		},
	}
}
