// Program entry point: assembly and startup only, no business logic.
package main

import (
	"log"
	"net/http"

	"<% .Module %>/internal/server"
<%if .Options.sqlite%>	"<% .Module %>/internal/store"
<%end%>)

var version = "dev" // default when not injected; release builds inject via -ldflags "-X main.version="

func main() {
<%if .Options.sqlite%>	db, err := store.Open(store.DefaultDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

<%end%>	handler := server.New(<%if .Options.sqlite%>db<%end%>)
	log.Printf("listening on :8080 (version %s)", version)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
