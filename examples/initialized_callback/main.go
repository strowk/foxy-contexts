package main

import (
	"log"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// This example defines an MCP Server with 'intialized' callback
// , that demonstrates how you can plug into server lifecycle and do
// some workafter the client has finished initialization and notified the server
// , run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:6274
// , then click Connect

// --8<-- [start:server]
func main() {
	log.Printf("starting server")
	app.
		NewBuilder().
		// this bit is WIP for now:

		// WithInitializedFinishedHandler(func(
		// 	shutdowner fx.Shutdowner,
		// ) server.ServerOption {
		// 	return &server.InitializationFininshedHandlerOption{
		// 		Callback: func(req *mcp.InitializedNotification) {
		// 			// do some work after client has finished initialization

		// 			// this is tad strange, but we would like to simply shut down when client is ready...
		// 			// why not, maybe we just want to test something and then shut down?

		// 			// Note: This currently is causing deadlock, as we are inside of input handling routine
		// 			// , which cannot finish as it waits for us here, where we wait for graceful shutdown,
		// 			// which in turn waits for input handling routine to finish, so.. essentially asking
		// 			// for shutdown anywhere that is processed within server's Handle method, would always
		// 			// cause this to stuck.

		// 			log.Printf("asking fx to shut down")
		// 			shutdowner.Shutdown()
		// 		},
		// 	}
		// }).
		// setting up server
		WithName("example-server").
		WithVersion("0.0.1").
		WithTransport(stdio.NewTransport()).
		// Configuring fx logging to only show errors
		WithFxOptions(
			fx.Provide(func() *zap.Logger {
				cfg := zap.NewDevelopmentConfig()
				cfg.Level.SetLevel(zap.ErrorLevel)
				logger, _ := cfg.Build()
				return logger
			}),
			fx.Option(fx.WithLogger(
				func(logger *zap.Logger) fxevent.Logger {
					return &fxevent.ZapLogger{Logger: logger}
				},
			)),
		).Run()
}

// --8<-- [end:server]

func Ptr[T any](v T) *T {
	return &v
}
