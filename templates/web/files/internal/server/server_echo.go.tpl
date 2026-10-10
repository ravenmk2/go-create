// Package server assembles HTTP routes and handlers.
package server

import (
<%if .Options.sqlite%>	"database/sql"
<%end%>	"net/http"

	"github.com/labstack/echo/v4"
<%if .Options.embed_frontend%>
	"<% .Module %>/web"
<%end%>)

// New assembles routes and returns the root HTTP handler.
func New(<%if .Options.sqlite%>db *sql.DB<%end%>) http.Handler {
	e := echo.New()
	e.GET("/healthz", func(c echo.Context) error {
<%if .Options.sqlite%>		if err := db.PingContext(c.Request().Context()); err != nil {
			return c.String(http.StatusServiceUnavailable, "db unavailable")
		}
<%end%>		return c.String(http.StatusOK, "ok")
	})
<%if .Options.embed_frontend%>
	// SPA fallback: wildcard route falls back to index.html for the frontend router
	e.GET("/*", echo.WrapHandler(SPAHandler(web.DistFS())))
<%end%>	return e
}
