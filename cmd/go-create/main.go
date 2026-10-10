package main

import "github.com/ravenmk2/go-create/internal/cli"

var version = "dev"

func main() {
	cli.Execute(version)
}
