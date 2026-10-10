// Package server assembles HTTP routes and handlers.
package server

import (
<%if .Options.sqlite%>	"database/sql"
<%end%>	"net/http"

	"github.com/gin-gonic/gin"
<%if .Options.embed_frontend%>
	"<% .Module %>/web"
<%end%>)

// New assembles routes and returns the root HTTP handler.
func New(<%if .Options.sqlite%>db *sql.DB<%end%>) http.Handler {
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
<%if .Options.sqlite%>		if err := db.PingContext(c.Request.Context()); err != nil {
			c.String(http.StatusServiceUnavailable, "db unavailable")
			return
		}
<%end%>		c.String(http.StatusOK, "ok")
	})
<%if .Options.embed_frontend%>
	// SPA fallback: unmatched routes fall back to index.html for the frontend router
	r.NoRoute(gin.WrapH(SPAHandler(web.DistFS())))
<%end%>	return r
}
