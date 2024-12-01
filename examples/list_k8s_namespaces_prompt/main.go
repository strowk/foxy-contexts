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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// This example defines list-k8s-contexts prompt for MCP server, that uses k8s client-go to list available namespaces
	// and returns them as a response, run it with:
	// npx @modelcontextprotocol/inspector go run main.go
	// , then in browser open http://localhost:5173
	// , then click Connect
	// , then go to Prompts
	// , then click List Prompts
	// , then click list-k8s-contexts
	// , then click Get Prompt

	fx.New(
		// Here we define a prompt that lists k8s contexts using client-go
		fx.Provide(fxctx.AsPrompt(
			func() fxctx.Prompt {

				return fxctx.NewPrompt(
					// This information about the prompt would be used when it is listed:
					mcp.Prompt{
						Name:        "list-k8s-namespaces",
						Description: Ptr("List Kubernetes namespaces"),
						Arguments: []mcp.PromptArgument{
							{
								Description: Ptr("Kubernetes context"),
								Name:        "context",
								Required:    Ptr(true),
							},
						},
					},

					// This is the callback that would be executed when the prompt/get is requested:
					func(req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
						k8sContext := req.Params.Arguments["context"]
						loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
						configOverrides := &clientcmd.ConfigOverrides{}

						configOverrides.CurrentContext = k8sContext

						kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
							loadingRules,
							configOverrides,
						)

						config, err := kubeConfig.ClientConfig()
						if err != nil {
							log.Printf("failed to get kubeconfig: %v", err)
							return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
						}

						clientset, err := kubernetes.NewForConfig(config)
						if err != nil {
							log.Printf("failed to create k8s client: %v", err)
							return nil, fmt.Errorf("failed to create k8s client: %w", err)
						}

						ns, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})

						if err != nil {
							log.Printf("failed to list namespaces: %v", err)
							return nil, fmt.Errorf("failed to list namespaces: %w", err)
						}

						var namespaces []struct {
							Namespace string `json:"namespace"`
						} = make([]struct {
							Namespace string `json:"namespace"`
						}, len(ns.Items))

						for i, n := range ns.Items {

							namespaces[i] = struct {
								Namespace string `json:"namespace"`
							}{
								Namespace: n.Name,
							}
						}

						marshalled, err := json.Marshal(namespaces)
						if err != nil {
							log.Printf("failed to marshal namespace: %v", err)
							return nil, fmt.Errorf("failed to marshal namespace: %w", err)
						}

						return &mcp.GetPromptResult{
							Meta:        map[string]interface{}{},
							Description: Ptr(fmt.Sprintf("Namespaces in context %s", k8sContext)),
							Messages: []mcp.PromptMessage{
								{
									Content: mcp.TextContent{
										Type: "text",
										Text: string(marshalled),
									},
									Role: mcp.RoleAssistant,
								},
							},
						}, nil
					},
				)
			},
		)),

		// PromptMux registers prompts and provides them to the server for listing and getting
		fxctx.ProvidePromptMux(),

		// Start the server using stdio transport
		fx.Invoke((func(
			lc fx.Lifecycle,
			promptMux fxctx.PromptMux,
		) {
			transport := stdio.NewTransport()
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						transport.Run(
							mcp.ServerCapabilities{
								Prompts: &mcp.ServerCapabilitiesPrompts{
									ListChanged: Ptr(false),
								},
							},
							mcp.Implementation{
								Name:    "my-mcp-k8s-server",
								Version: "0.0.1",
							},
							server.ServerStartCallbackOption{
								Callback: func(s server.Server) {
									// This makes sure that server is aware of the prompts
									// we have registered and both can list and call them
									promptMux.RegisterHandlers(s)
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
