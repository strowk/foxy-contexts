package main

import (
	"context"
	"fmt"
	"os"

	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	// This example defines list-current-dir-files tool for MCP server, that prints files in the current directory
	// , run it with:
	// npx @modelcontextprotocol/inspector go run main.go
	// , then in browser open http://localhost:5173
	// , then click Connect
	// , then click List Tools
	// , then click list-current-dir-files

	fx.New(
		// Here we define a tool that lists k8s contexts using client-go
		fx.Provide(fxctx.AsTool(
			func() fxctx.Tool {

				return fxctx.NewTool(
					// This information about the tool would be used when it is listed:
					"list-current-dir-files",
					"Lists files in the current directory",
					mcp.ToolInputSchema{
						Type:       "object",
						Properties: map[string]map[string]interface{}{},
						Required:   []string{},
					},

					// This is the callback that would be executed when the tool is called:
					func(args map[string]interface{}) fxctx.ToolResponse {
						files, err := os.ReadDir(".")
						if err != nil {
							return fxctx.ToolResponse{
								IsError: Ptr(true),
								Meta:    map[string]interface{}{},
								Content: []interface{}{
									mcp.TextContent{
										Type: "text",
										Text: fmt.Sprintf("failed to read dir: %v", err),
									},
								},
							}
						}
						var contents []interface{} = make([]interface{}, len(files))
						for i, f := range files {
							contents[i] = mcp.TextContent{
								Type: "text",
								Text: f.Name(),
							}
						}

						return fxctx.ToolResponse{
							Meta:    map[string]interface{}{},
							Content: contents,
							IsError: Ptr(false),
						}
					},
				)
			},
		)),

		// ToolMux registers tools and provides them to the server for listing tools and calling them
		fxctx.ProvideToolMux(),

		// Start the server using stdio transport
		fx.Invoke((func(
			lc fx.Lifecycle,
			toolMux fxctx.ToolMux,
		) {
			transport := stdio.NewTransport()
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						transport.Run(
							mcp.ServerCapabilities{
								Tools: &mcp.ServerCapabilitiesTools{
									ListChanged: Ptr(false),
								},
							},
							mcp.Implementation{
								Name:    "my-mcp-server",
								Version: "0.0.1",
							},
							server.ServerStartCallbackOption{
								Callback: func(s server.Server) {
									// This makes sure that server is aware of the tools
									// we have registered and both can list and call them
									toolMux.RegisterHandlers(s)
								},
							},
						)
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return transport.Shutdown(ctx)
				},
			})
		})),

		// Just configuring fx logging to only show errors
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

func Ptr[T any](v T) *T {
	return &v
}
