package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// This example defines list-k8s-contexts tool for MCP server, that uses k8s client-go to list available contexts
// and returns them as a response, run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:5173
// , then click Connect
// , then click List Tools
// , then click list-k8s-contexts

func NewListK8sContextsTool() fxctx.Tool {
	// Here we define a tool that lists k8s contexts using client-go

	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "list-k8s-contexts",
			Description: Ptr("List Kubernetes contexts from configuration files such as kubeconfig"),
			InputSchema: mcp.ToolInputSchema{
				Type:       "object",
				Properties: map[string]map[string]interface{}{},
				Required:   []string{},
			},
		},

		// This is the callback that would be executed when the tool is called:
		func(args map[string]interface{}) *mcp.CallToolResult {
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
			cfg, err := kubeConfig.RawConfig()
			if err != nil {
				log.Printf("failed to get kubeconfig: %v", err)
				return &mcp.CallToolResult{
					IsError: Ptr(true),
					Meta:    map[string]interface{}{},
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("failed to get kubeconfig: %v", err),
						},
					},
				}
			}

			return &mcp.CallToolResult{
				Meta:    map[string]interface{}{},
				Content: getListContextsToolContent(cfg),
				IsError: Ptr(false),
			}
		},
	)
}

func main() {
	fx.New(
		// Registering the tool with fx
		fx.Provide(fxctx.AsTool(NewListK8sContextsTool)),

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
							&mcp.ServerCapabilities{
								Tools: &mcp.ServerCapabilitiesTools{
									ListChanged: Ptr(false),
								},
							},
							&mcp.Implementation{
								Name:    "my-mcp-k8s-server",
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

func getListContextsToolContent(cfg api.Config) []interface{} {
	var contents []interface{} = make([]interface{}, len(cfg.Contexts))
	i := 0
	for _, c := range cfg.Contexts {
		marshalled, err := json.Marshal(ContextJsonEncoded{
			Context: c,
			Name:    c.Cluster,
		})
		if err != nil {
			log.Printf("failed to marshal context: %v", err)
			continue
		}
		contents[i] = mcp.TextContent{
			Type: "text",
			Text: string(marshalled),
		}
		i++
	}
	return contents
}

type ContextJsonEncoded struct {
	Context *api.Context `json:"context"`
	Name    string       `json:"name"`
}

func Ptr[T any](v T) *T {
	return &v
}
