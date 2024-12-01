# Foxy Contexts

Foxy contexts is a library for building context servers supporting [Model Context Protocol](https://modelcontextprotocol.io/).

This library only supports server side of the protocol. Using it you can build context servers using declarative approach, by defining [tools](https://modelcontextprotocol.io/docs/concepts/tools), [resources](https://modelcontextprotocol.io/docs/concepts/resources) and [prompts](https://modelcontextprotocol.io/docs/concepts/prompts) and then registering them within DI using [uber's fx](https://modelcontextprotocol.io/docs/concepts/resources).

With this approach you can easily colocate call/read/get logic and definitions of your tools/resources/prompts in a way that every tool/resource/prompt is placed in a separate place, but related code is colocated.

## Try it out

Check examples in [examples](./examples) directory.

Fox example:

```bash
cd examples/list_k8s_contexts_tool
npx @modelcontextprotocol/inspector go run main.go
```
, then in browser open http://localhost:5173 and try to use list-k8s-contexts tool, then check out implementation in [examples/list_k8s_contexts_tool/main.go](./examples/list_k8s_contexts_tool/main.go).

Here's the code of that example:


```go
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

func main() {
	// This example defines list-k8s-contexts tool for MCP server, that uses k8s client-go to list available contexts
	// and returns them as a response, run it with:
	// npx @modelcontextprotocol/inspector go run main.go
	// , then in browser open http://localhost:5173
	// , then click Connect
	// , then click List Tools
	// , then click list-k8s-contexts

	fx.New(
		// Here we define a tool that lists k8s contexts using client-go
		fx.Provide(fxctx.AsTool(
			func() fxctx.Tool {

				return fxctx.NewTool(
					// This information about the tool would be used when it is listed:
					"list-k8s-contexts",
					"List Kubernetes contexts from configuration files such as kubeconfig",
					mcp.ToolInputSchema{
						Type:       "object",
						Properties: map[string]map[string]interface{}{},
						Required:   []string{},
					},

					// This is the callback that would be executed when the tool is called:
					func(args map[string]interface{}) fxctx.ToolResponse {
						loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
						kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
						cfg, err := kubeConfig.RawConfig()
						if err != nil {
							log.Printf("failed to get kubeconfig: %v", err)
							return fxctx.ToolResponse{
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

						return fxctx.ToolResponse{
							Meta:    map[string]interface{}{},
							Content: getListContextsToolContent(cfg),
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
```




