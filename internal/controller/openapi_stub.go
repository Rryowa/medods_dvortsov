package controller

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Reason string `json:"reason"`
}

func GetSwagger() (*openapi3.T, error) {
	return &openapi3.T{}, nil
}

func RegisterHandlersWithBaseURL(e *echo.Echo, c *Controller, base string) {
	g := e.Group(base)
	g.GET("/ping", c.CheckServer)

	// Auth routes
	ag := g.Group("/auth")
	ag.POST("/token/issue", c.IssueTokens)
	ag.POST("/token/refresh", c.RefreshTokens)
	ag.POST("/logout", c.Logout)
	ag.GET("/user/guid", c.GetUserGUID)
}
