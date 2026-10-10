// Command <% .Name %> is the program entrypoint; it only wires up and starts the application, without business logic.
package main

var version = "dev" // default value when not injected at link time

func main() {
	Execute()
}
