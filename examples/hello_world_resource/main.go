package main

import (
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

// --8<-- [start:tool]
func NewGreatResource() fxctx.Resource {
	return fxctx.NewResource(
		mcp.Resource{
			Name:        "hello-world",
			Uri:         "hello-world://hello-world",
			MimeType:    Ptr("application/json"),
			Description: Ptr("Hello World Resource"),
			Annotations: &mcp.ResourceAnnotations{
				Audience: []mcp.Role{
					mcp.RoleAssistant, mcp.RoleUser,
				},
			},
		},
		func(uri string) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []interface{}{
					mcp.TextResourceContents{
						MimeType: Ptr("application/json"),
						Text:     `{"hello": "world"}`,
						Uri:      uri,
					},
				},
			}, nil
		},
	)
}

// --8<-- [end:tool]

// --8<-- [start:server]
func main() {
	err := app.
		NewBuilder().
		// adding the resource to the app
		WithResource(NewGreatResource).
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
