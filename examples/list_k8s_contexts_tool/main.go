package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"github.com/strowk/foxy-contexts/pkg/toolinput"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// This example defines list-k8s-contexts tool for MCP server, that uses k8s client-go to list available contexts
// and returns them as a response, run it with:
// npx @modelcontextprotocol/inspector go run .
// , then in browser open http://localhost:6274
// , then click Connect
// , then click List Tools
// , then click list-k8s-contexts

// --8<-- [start:dependency_inject]
func NewListK8sContextsTool(kubeConfig clientcmd.ClientConfig) fxctx.Tool { // --8<-- [end:dependency_inject]

	// --8<-- [start:toolinput]
	schema := toolinput.NewToolInputSchema(
		toolinput.WithString("kubeconfig", "Path to kubeconfig file"),
	)

	// Here we define a tool that lists k8s contexts using client-go
	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "list-k8s-contexts",
			Description: utils.Ptr("List Kubernetes contexts from configuration files such as kubeconfig"),
			InputSchema: schema.GetMcpToolInputSchema(),
		},

		// This is the callback that would be executed when the tool is called:
		func(_ context.Context, args map[string]interface{}) *mcp.CallToolResult {
			input, err := schema.Validate(args)
			if err != nil {
				log.Printf("failed to validate input: %v", err)
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("failed to validate input: %v", err),
						},
					},
				}
			}

			path := input.StringOr("kubeconfig", "")
			if path != "" {
				os.Setenv("KUBECONFIG", path)
			}

			// --8<-- [end:toolinput]

			cfg, err := kubeConfig.RawConfig()
			if err != nil {
				log.Printf("failed to get kubeconfig: %v", err)
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
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
				IsError: utils.Ptr(false),
			}
		},
	)
}

func main() {
	// --8<-- [start:dependency_provide]
	app.
		NewBuilder().
		WithFxOptions(
			fx.Provide(NewK8sClientConfig),
		).
		WithTool(NewListK8sContextsTool). // --8<-- [end:dependency_provide]
		WithServerCapabilities(&mcp.ServerCapabilities{
			Tools: &mcp.ServerCapabilitiesTools{
				ListChanged: utils.Ptr(false),
			},
		}).
		WithTransport(stdio.NewTransport()).
		WithName("list-k8s-contexts-tool").
		WithVersion("0.0.1").
		WithFxOptions(
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
		).
		Run()
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
