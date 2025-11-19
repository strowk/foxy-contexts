package main

import (
	"context"
	"log"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
)

// This example defines resource tool for MCP server
// , run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:6274
// , then click Connect
// , then click List Resources
// , then click hello-world

// --8<-- [start:resource]
func NewGreatResource() fxctx.Resource {
	return fxctx.NewResource(
		mcp.Resource{
			Name:        "hello-world",
			Uri:         "hello-world://hello-world",
			MimeType:    utils.Ptr("application/json"),
			Description: utils.Ptr("Hello World Resource"),
			Annotations: &mcp.ResourceAnnotations{
				Audience: []mcp.Role{
					mcp.RoleAssistant, mcp.RoleUser,
				},
			},
		},
		func(_ context.Context, uri string) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []interface{}{
					mcp.TextResourceContents{
						MimeType: utils.Ptr("application/json"),
						Text:     `{"hello": "world"}`,
						Uri:      uri,
					},
				},
			}, nil
		},
	)
}

// --8<-- [end:resource]

// --8<-- [start:server]
func main() {
	err := app.
		NewBuilder().
		// adding the resource to the app
		WithResource(NewGreatResource).
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
		Run()
	if err != nil {
		log.Fatal(err)
	}
}

// --8<-- [end:server]
