package main

import (
	"context"
	"fmt"
	"log"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// This example defines resource tool for MCP server
// , run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:5173
// , then click Connect
// , then click List Resources
// , then click hello-world

// --8<-- [start:provider]
func NewGreatResourceProvider() fxctx.ResourceProvider {
	return fxctx.NewResourceProvider(
		// This is the callback that would be executed when the resources/list is requested:
		func(_ context.Context) ([]mcp.Resource, error) {
			return []mcp.Resource{
				{
					Name:        "my-great-resource-one",
					Description: Ptr("Does something great"),
					Uri:         "/resources/great/one",
				},
			}, nil
		},
		//  This function reads the resource for a given uri to run when resources/read is requested:
		func(_ context.Context, uri string) (*mcp.ReadResourceResult, error) {

			// you would probably be doing something more complicated here
			// like reading from a database or calling an external service
			// based on what you have parsed from the uri
			if uri == "/resources/great/one" {
				return &mcp.ReadResourceResult{
					Contents: []interface{}{
						mcp.TextResourceContents{
							MimeType: Ptr("application/json"),
							Text:     string(`{"great": "resource"}`),
							Uri:      uri,
						},
					},
				}, nil
			}

			// this error would be wrapped in JSON-RPC error response
			return nil, fmt.Errorf("resource not found")
		},
	)
}

// --8<-- [end:provider]

// --8<-- [start:server]
func main() {
	err := app.
		NewBuilder().
		// adding the resource provider to server
		WithResourceProvider(NewGreatResourceProvider).
		// setting up server
		WithName("my-mcp-server").
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

	if err != nil {
		log.Fatal(err)
	}
}

// --8<-- [end:server]

func Ptr[T any](v T) *T {
	return &v
}
