package main

import (
	"context"
	"fmt"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// This example defines my-great-tool tool for MCP server
// , run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:6274
// , then click Connect
// , then click List Tools
// , then click my-great-tool

// --8<-- [start:tool]
func NewGreatTool() fxctx.Tool {
	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "my-great-tool",
			Description: utils.Ptr("The great tool"),
			InputSchema: mcp.ToolInputSchema{ // here we tell client what we expect as input
				Type:       "object",
				Properties: map[string]map[string]interface{}{},
				Required:   []string{},
			},
		},

		// This is the callback that would be executed when the tool is called:
		func(_ context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// here we can do anything we want
			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Sup"),
					},
				},
			}
		},
	)
}

// --8<-- [end:tool]

// --8<-- [start:server]
func main() {
	app.
		NewBuilder().
		// adding the tool to the app
		WithTool(NewGreatTool).
		WithServerCapabilities(&mcp.ServerCapabilities{
			Tools: &mcp.ServerCapabilitiesTools{
				ListChanged: utils.Ptr(false),
			},
		}).
		// setting up server
		WithName("great-tool-server").
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
