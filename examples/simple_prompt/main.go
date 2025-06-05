package main

import (
	"context"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// --8<-- [start:prompt]
func NewGreatPrompt() fxctx.Prompt {
	return fxctx.NewPrompt(
		// This information about the prompt would be used when it is listed:
		mcp.Prompt{
			Name:        "my-great-prompt",
			Description: utils.Ptr("Doing something great"),
			Arguments: []mcp.PromptArgument{
				{
					Description: utils.Ptr("An argument for the prompt"),
					Name:        "arg-1",
					Required:    utils.Ptr(true),
				},
			},
		},
		// This is the callback that would be executed when the prompt/get is requested:
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			description := "Prompting to do something great"
			return &mcp.GetPromptResult{
				Description: &description,
				Messages: []mcp.PromptMessage{
					{
						Content: mcp.TextContent{
							Type: "text",
							Text: "Would you like to do something great?",
						},
						Role: mcp.RoleUser,
					},
				},
			}, nil
		},
	)
}

// --8<-- [end:prompt]

// --8<-- [start:server]
func main() {
	app.
		NewBuilder().
		// adding the tool to the app
		WithPrompt(NewGreatPrompt).
		WithServerCapabilities(&mcp.ServerCapabilities{
			Prompts: &mcp.ServerCapabilitiesPrompts{
				ListChanged: utils.Ptr(false),
			},
		}).
		// setting up server
		WithName("great-tool-server").
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
}

// --8<-- [end:server]
