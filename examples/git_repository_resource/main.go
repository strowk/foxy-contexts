package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	// This example defines resource tool for MCP server, that shows information about a current git repository
	// , run it with:
	// npx @modelcontextprotocol/inspector go run main.go
	// , then in browser open http://localhost:5173
	// , then click Connect
	// , then click List Resources
	// , then click current-git-repository

	err := app.
		NewBuilder().
		// adding the resource to the app
		WithResource(func() fxctx.Resource {
			return fxctx.NewResource(
				mcp.Resource{
					Name:        "current-git-repository",
					Uri:         "git://current-git-repository",
					MimeType:    Ptr("application/json"),
					Description: Ptr("Shows information about a current git repository"),
					Annotations: &mcp.ResourceAnnotations{
						Audience: []mcp.Role{
							mcp.RoleAssistant, mcp.RoleUser,
						},
					},
				},
				func(uri string) (*mcp.ReadResourceResult, error) {
					repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
						DetectDotGit: true,
					})
					if err != nil {
						log.Printf("failed to open git repository: %v", err)
						return nil, err
					}

					type remote struct {
						Name string   `json:"name"`
						Urls []string `json:"urls"`
					}
					gitRemotes, err := repo.Remotes()
					if err != nil {
						log.Printf("failed to get remotes: %v", err)
						return nil, fmt.Errorf("failed to get remotes: %w", err)
					}

					var remotes []remote = make([]remote, len(gitRemotes))
					for i, r := range gitRemotes {
						remotes[i] = remote{
							Name: r.Config().Name,
							Urls: r.Config().URLs,
						}
					}

					data, err := json.Marshal(remotes)
					if err != nil {
						log.Printf("failed to marshal remotes: %v", err)
						return nil, fmt.Errorf("failed to marshal remotes: %w", err)
					}

					return &mcp.ReadResourceResult{
						Contents: []interface{}{
							mcp.TextResourceContents{
								MimeType: Ptr("application/json"),
								Text:     string(data),
								Uri:      uri,
							},
						},
					}, nil
				},
			)
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

func Ptr[T any](v T) *T {
	return &v
}
