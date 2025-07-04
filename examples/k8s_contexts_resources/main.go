package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/strowk/foxy-contexts/internal/utils"
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
		func(_ context.Context) ([]mcp.Resource, error) {
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
		func(_ context.Context, uri string) (*mcp.ReadResourceResult, error) {
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
					MimeType: utils.Ptr("application/json"),
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
		WithServerCapabilities(&mcp.ServerCapabilities{
			Resources: &mcp.ServerCapabilitiesResources{
				ListChanged: utils.Ptr(false),
				Subscribe:   utils.Ptr(false),
			},
		}).
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

func toMcpResourcse(contextName string) mcp.Resource {
	return mcp.Resource{
		Annotations: &mcp.ResourceAnnotations{
			Audience: []mcp.Role{mcp.RoleAssistant, mcp.RoleUser},
		},
		Name:        contextName,
		Description: utils.Ptr("Specific k8s context as read from kubeconfig configuration files"),
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
			MimeType: utils.Ptr("application/json"),
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
