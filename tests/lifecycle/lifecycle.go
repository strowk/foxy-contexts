package main

import (
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/stdio"
)

func main() {
	app.
		NewBuilder().
		// setting up server
		WithName("great-tool-server").
		WithVersion("0.0.1").
		WithTransport(stdio.NewTransport()).
		Run()
}
