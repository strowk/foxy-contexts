package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// This example defines k8s contexts as resources for MCP server,
// that uses k8s client-go to provide contexts as resources

func NewK8sContextsResourceProvider() fxctx.ResourceProvider {
	return fxctx.NewResourceProvider(

		// This function returns list of resources in order to include them to the list of resources
		func() ([]mcp.Resource, error) {
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
			cfg, err := kubeConfig.RawConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
			}

			resources := []mcp.Resource{}
			for name := range cfg.Contexts {
				resources = append(resources, toMcpResourcse(name))
			}
			return resources, nil
		},

		//  This function reads the resource for a given uri
		func(uri string) (*mcp.ReadResourceResult, error) {
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
			cfg, err := kubeConfig.RawConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
			}

			if uri == "contexts" {
				contents := getContextsContent(uri, cfg)
				return &mcp.ReadResourceResult{
					Contents: contents,
				}, nil
			}

			if strings.HasPrefix(uri, "contexts/") {
				name := strings.TrimPrefix(uri, "contexts/")
				c, ok := cfg.Contexts[name]
				if !ok {
					return nil, fmt.Errorf("context not found: %s", name)
				}

				var contents []interface{} = make([]interface{}, 1)
				marshalled, err := json.Marshal(&struct {
					Context *api.Context `json:"context"`
					Name    string       `json:"name"`
				}{Context: c, Name: name})
				if err != nil {
					return nil, fmt.Errorf("failed to marshal context: %w", err)
				}

				contents[0] = mcp.TextResourceContents{
					MimeType: Ptr("application/json"),
					Text:     string(marshalled),
					Uri:      uri,
				}

				return &mcp.ReadResourceResult{
					Contents: contents,
				}, nil
			}

			return nil, fmt.Errorf("unknown uri: %s", uri)
		})

}

func main() {

	err := app.
		NewBuilder().
		WithResourceProvider(NewK8sContextsResourceProvider).
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

	// fx.New(
	// 	// Here we define a resource provider that provides k8s contexts as resources
	// 	fx.Provide(fxctx.AsResourceProvider()),

	// 	// ResourceMux registers resources and resource providers and serves that data for listing resources and reading them
	// 	fxctx.ProvideResourceMux(),

	// 	// ToolMux registers tools and provides them to the server for listing tools and calling them
	// 	// TODO: this normally does not need to be here, we should teach server to provide empty list of tools
	// 	// without actually registering the tool mux?
	// 	fxctx.ProvideToolMux(),

	// 	// Start the server using stdio transport
	// 	fx.Invoke((func(
	// 		lc fx.Lifecycle,
	// 		resourceMux fxctx.ResourceMux,
	// 		toolsMux fxctx.ToolMux,
	// 	) {
	// 		transport := stdio.NewTransport()
	// 		lc.Append(fx.Hook{
	// 			OnStart: func(ctx context.Context) error {
	// 				go func() {
	// 					transport.Run(
	// 						&mcp.ServerCapabilities{
	// 							Resources: &mcp.ServerCapabilitiesResources{
	// 								ListChanged: Ptr(false),
	// 								Subscribe:   Ptr(false),
	// 							},
	// 						},
	// 						&mcp.Implementation{
	// 							Name:    "my-mcp-k8s-server",
	// 							Version: "0.0.1",
	// 						},
	// 						server.ServerStartCallbackOption{
	// 							Callback: func(s server.Server) {
	// 								// This makes sure that server is aware of the tools
	// 								// we have registered and both can list and call them
	// 								resourceMux.RegisterHandlers(s)
	// 								toolsMux.RegisterHandlers(s)
	// 							},
	// 						},
	// 					)
	// 				}()
	// 				return nil
	// 			},
	// 			OnStop: func(ctx context.Context) error {
	// 				return transport.Shutdown(ctx)
	// 			},
	// 		})
	// 	})),

	// 	// Just configuring fx logging to only show errors
	// 	fx.Provide(func() *zap.Logger {
	// 		cfg := zap.NewDevelopmentConfig()
	// 		cfg.Level.SetLevel(zap.ErrorLevel)
	// 		logger, _ := cfg.Build()
	// 		return logger
	// 	}),
	// 	fx.Option(fx.WithLogger(
	// 		func(logger *zap.Logger) fxevent.Logger {
	// 			return &fxevent.ZapLogger{Logger: logger}
	// 		},
	// 	)),
	// ).Run()

}

func toMcpResourcse(contextName string) mcp.Resource {
	return mcp.Resource{Annotations: &mcp.ResourceAnnotations{
		Audience: []mcp.Role{mcp.RoleAssistant, mcp.RoleUser},
	},
		Name:        contextName,
		Description: Ptr("Specific k8s context as read from kubeconfig configuration files"),
		Uri:         "contexts/" + contextName,
	}
}

func getContextsContent(uri string, cfg api.Config) []interface{} {
	var contents []interface{} = make([]interface{}, len(cfg.Contexts))
	i := 0

	for name, c := range cfg.Contexts {
		marshalled, err := json.Marshal(c)
		if err != nil {
			log.Printf("failed to marshal context: %v", err)
			continue
		}

		contents[i] = mcp.TextResourceContents{
			MimeType: Ptr("application/json"),
			Text:     string(marshalled),
			Uri:      uri + "/" + name,
		}
		i++
	}
	return contents
}

type ContextContent struct {
	Uri     string       `json:"uri"`
	Text    string       `json:"text"`
	Context *api.Context `json:"context"`
	Name    string       `json:"name"`
}

func Ptr[T any](v T) *T {
	return &v
}
