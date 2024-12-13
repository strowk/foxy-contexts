package app

import (
	"context"
	"errors"

	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

var (
	ErrNoTransportSpecified = errors.New("no transport specified, please use WithTransport to specify a transport")
)

func NewFoxyApp() *FoxyApp {
	return &FoxyApp{
		implementation: &mcp.Implementation{
			Name:    "my-foxy-contexts-server",
			Version: "0.0.1",
		},
	}
}

// FoxyApp is essentially a builder for a foxy app, that wraps fx.App
// and provides a more user-friendly interface for building and running
// your MCP server
//
// You would be calling WithTool, WithResource, WithResourceProvider, WithPrompt
// to register your tools, resources, resource providers and prompts and then
// calling Run to start the server, or you can instead call BuildFxApp to get the
// fx.App instance and run it yourself. You must set transport using
// WithTransport. Unless you configure server using
// WithName and WithVersion, it will use default values "my-foxy-contexts-server" and "0.0.1".
// Finally you can use WithFxOptions to pass additional fx.Options to the fx.App instance
// before it is built.
type FoxyApp struct {
	implementation *mcp.Implementation
	transport      server.Transport

	options []fx.Option
}

// WithTool adds a tool to the app
//
// newTool must be a function that returns a fxctx.Tool
// it can also take in any dependencies that you want to inject
// into the tool, that will be resolved by the fx framework
func (f *FoxyApp) WithTool(newTool any) *FoxyApp {
	f.options = append(f.options, fx.Provide(fxctx.AsTool(newTool)))
	return f
}

// WithResource adds a resource to the app
//
// newResource must be a function that returns a fxctx.Resource
// it can also take in any dependencies that you want to inject
// into the resource, that will be resolved by the fx framework
func (f *FoxyApp) WithResource(newResource any) *FoxyApp {
	f.options = append(f.options, fx.Provide(fxctx.AsResource(newResource)))
	return f
}

// WithResourceProvider adds a resource provider to the app
//
// newResourceProvider must be a function that returns a fxctx.ResourceProvider
// it can also take in any dependencies that you want to inject
// into the resource provider, that will be resolved by the fx framework
func (f *FoxyApp) WithResourceProvider(newResourceProvider any) *FoxyApp {
	f.options = append(f.options, fx.Provide(fxctx.AsResourceProvider(newResourceProvider)))
	return f
}

// WithPrompt adds a prompt to the app
//
// newPrompt must be a function that returns a fxctx.Prompt
// it can also take in any dependencies that you want to inject
// into the prompt, that will be resolved by the fx framework
func (f *FoxyApp) WithPrompt(newPrompt any) *FoxyApp {
	f.options = append(f.options, fx.Provide(fxctx.AsPrompt(newPrompt)))
	return f
}

// WithStdioTransport sets up the server to use stdio transport
//
// options can be used to configure the stdio transport
func (f *FoxyApp) WithTransport(transport server.Transport) *FoxyApp {
	f.transport = transport
	return f
}

// WithFxOptions adds additional fx.Options to fx.App instance
func (f *FoxyApp) WithFxOptions(opts ...fx.Option) *FoxyApp {
	f.options = append(f.options, fx.Options(opts...))
	return f
}

// WithName sets the name of the server
//
// The name would be returned to client during the initialization
// phase, if not set, it would default to "my-foxy-contexts-server"
func (f *FoxyApp) WithName(name string) *FoxyApp {
	f.implementation.Name = name
	return f
}

// WithVersion sets the version of the server
//
// The version would be returned to client during the initialization
// phase, if not set, it would default to "0.0.1"
func (f *FoxyApp) WithVersion(version string) *FoxyApp {
	f.implementation.Version = version
	return f
}

// BuildFxApp builds the fx.App instance as configured by `With*` methods
func (f *FoxyApp) BuildFxApp() (*fx.App, error) {
	f.options = append(f.options, fxctx.ProvideToolMux())
	f.options = append(f.options, fxctx.ProvideResourceMux())
	f.options = append(f.options, fxctx.ProvidePromptMux())
	f.options = append(f.options, fxctx.ProvideCompleteMux())

	if f.transport == nil {
		return nil, ErrNoTransportSpecified
	}

	f.options = append(f.options, f.provideServerLifecycle(f.transport))

	return fx.New(fx.Options(f.options...)), nil
}

// Run builds and runs the fx.App instance as configured by `With*` methods
func (f *FoxyApp) Run() error {
	app, err := f.BuildFxApp()
	if err != nil {
		return err
	}
	app.Run()
	return app.Err()
}

type ServerLifecycleParams struct {
	fx.In

	ToolMux     fxctx.ToolMux     `optional:"true"`
	ResourceMux fxctx.ResourceMux `optional:"true"`
	PromptMux   fxctx.PromptMux   `optional:"true"`
	CompleteMux fxctx.CompleteMux `optional:"true"`
}

func (f *FoxyApp) getServerCapabilities() *mcp.ServerCapabilities {
	serverCapabilities := &mcp.ServerCapabilities{}
	return serverCapabilities
}

func (f *FoxyApp) provideServerLifecycle(transport server.Transport) fx.Option {
	return fx.Invoke((func(
		lc fx.Lifecycle,
		p ServerLifecycleParams,
	) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					transport.Run(
						f.getServerCapabilities(),
						f.implementation,
						server.ServerStartCallbackOption{
							Callback: func(s server.Server) {
								if p.ToolMux != nil {
									p.ToolMux.RegisterHandlers(s)
								}
								if p.ResourceMux != nil {
									p.ResourceMux.RegisterHandlers(s)
								}
								if p.PromptMux != nil {
									p.PromptMux.RegisterHandlers(s)
								}
								if p.CompleteMux != nil {
									p.CompleteMux.RegisterHandlers(s)
								}
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
	}))
}
