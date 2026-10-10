// Command <% .Name %> is the program entrypoint; it only wires up and starts the application, without business logic.
package main

import (
	"flag"
	"fmt"
	"os"

	"<% .Module %>/internal/app"
)

var version = "dev" // default value when not injected at link time

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
