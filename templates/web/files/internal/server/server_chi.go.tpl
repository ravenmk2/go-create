// Package server assembles HTTP routes and handlers.
package server

import (
<%if .Options.sqlite%>	"database/sql"
<%end%>	"net/http"

	"github.com/go-chi/chi/v5"
<%if .Options.embed_frontend%>
	"<% .Module %>/web"
<%end%>)

// New assembles routes and returns the root HTTP handler.
func New(<%if .Options.sqlite%>db *sql.DB<%end%>) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
<%if .Options.sqlite%>		if err := db.PingContext(req.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
<%end%>		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
<%if .Options.embed_frontend%>
	// API routes must be registered before the SPA fallback to avoid being swallowed by it
	r.Get("/*", SPAHandler(web.DistFS()).ServeHTTP)
<%end%>	return r
}
