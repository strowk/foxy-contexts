package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/session"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type MySessionData struct {
	isItGreat bool
}

func (m *MySessionData) String() string {
	return "MySessionData"
}

// This example defines my-great-tool tool for MCP server that is using streamable http transport
// , run it with:
// go run main.go
// then in another terminal run:
// curl -X POST -H "Content-Type: application/json" -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' http://localhost:8080/mcp
// , then you should see the response:
// {"jsonrpc":"2.0","result":{"content":[{"text":"Sup, saving greatness to session","type":"text"}]},"id":0}

// --8<-- [start:tool]
func NewGreatTool(sm *session.SessionManager) fxctx.Tool {
	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "my-great-tool",
			Description: Ptr("The great tool"),
			InputSchema: mcp.ToolInputSchema{ // here we tell client what we expect as input
				Type:       "object",
				Properties: map[string]map[string]interface{}{},
				Required:   []string{},
			},
		},

		// This is the callback that would be executed when the tool is called:
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			data := sm.GetSessionData(ctx)
			if data == nil {
				sm.SetSessionData(ctx, &MySessionData{
					isItGreat: true,
				})
			}

			resp := "saving greatness to session"
			if data != nil {
				resp = "already great"
			}
			// here we can do anything we want
			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Sup, %s", resp),
					},
				},
			}
		},
	)
}

// --8<-- [end:tool]

// --8<-- [start:server]
func main() {
	server := app.
		NewBuilder().
		// adding the tool to the app
		WithTool(NewGreatTool).
		// setting up server
		WithName("great-tool-server").
		WithVersion("0.0.1").
		WithTransport(
			streamable_http.NewTransport(
				streamable_http.Endpoint{
					Hostname: "localhost",
					Port:     8080,
					Path:     "/mcp",
				}),
		).
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
		)

	err := server.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// --8<-- [end:server]

func Ptr[T any](v T) *T {
	return &v
}
