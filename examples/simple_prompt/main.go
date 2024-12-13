package main

import (
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
			Description: Ptr("Doing something great"),
			Arguments: []mcp.PromptArgument{
				{
					Description: Ptr("An argument for the prompt"),
					Name:        "arg-1",
					Required:    Ptr(true),
				},
			},
		},
		// This is the callback that would be executed when the prompt/get is requested:
		func(req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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
		NewFoxyApp().
		// adding the tool to the app
		WithPrompt(NewGreatPrompt).
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

func Ptr[T any](v T) *T {
	return &v
}