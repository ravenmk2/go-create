// Package server assembles HTTP routes and handlers.
package server

import (
<%if .Options.sqlite%>	"database/sql"
<%end%>	"net/http"
<%if .Options.embed_frontend%>
	"<% .Module %>/web"
<%end%>)

// New assembles routes and returns the root HTTP handler.
func New(<%if .Options.sqlite%>db *sql.DB<%end%>) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
<%if .Options.sqlite%>		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
<%end%>		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
<%if .Options.embed_frontend%>
	// API routes must be registered before the SPA fallback to avoid being swallowed by it
	mux.Handle("GET /", SPAHandler(web.DistFS()))
<%end%>	return mux
}
