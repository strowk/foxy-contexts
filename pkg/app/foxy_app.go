package app

import (
	"context"
	"errors"

	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/session"
	"go.uber.org/fx"
)

var (
	ErrNoTransportSpecified = errors.New("no transport specified, please use WithTransport to specify a transport")
)

func NewBuilder() *Builder {
	return &Builder{
		implementation: &mcp.Implementation{
			Name:    "my-foxy-contexts-server",
			Version: "0.0.1",
		},
	}
}

// Builder wraps fx.App and provides a more user-friendly interface for building
// and running your MCP server
//
// You would be calling WithTool, WithResource, WithResourceProvider, WithPrompt
// to register your tools, resources, resource providers and prompts and then
// calling Run to start the server, or you can instead call BuildFxApp to get the
// fx.App instance and run it yourself. You must set transport using
// WithTransport. Unless you configure server using
// WithName and WithVersion, it will use default values "my-foxy-contexts-server" and "0.0.1".
// Finally you can use WithFxOptions to pass additional fx.Options to the fx.App instance
// before it is built.
type Builder struct {
	implementation *mcp.Implementation
	transport      server.Transport

	capabilities *mcp.ServerCapabilities

	transportError error

	options []fx.Option

	extraServerOptions []server.ServerOption
}

// WithTool adds a tool to the app
//
// newTool must be a function that returns a fxctx.Tool
// it can also take in any dependencies that you want to inject
// into the tool, that will be resolved by the fx framework
func (f *Builder) WithTool(newTool ...any) *Builder {
	for i := range newTool {
		f.options = append(f.options, fx.Provide(fxctx.AsTool(newTool[i])))
	}
	return f
}

// WithServerCapabilities sets the server capabilities
//
// serverCapabilities is a struct that defines the capabilities of the server
// that would be returned to the client during the initialization phase
// if not set, it would default to an empty struct, which could cause
// clients to not work as expected, so it is recommended to set this
func (f *Builder) WithServerCapabilities(serverCapabilities *mcp.ServerCapabilities) *Builder {
	f.capabilities = serverCapabilities
	return f
}

// WithResource adds a resource to the app
//
// newResource must be a function that returns a fxctx.Resource
// it can also take in any dependencies that you want to inject
// into the resource, that will be resolved by the fx framework
func (f *Builder) WithResource(newResource ...any) *Builder {
	for i := range newResource {
		f.options = append(f.options, fx.Provide(fxctx.AsResource(newResource[i])))
	}
	return f
}

// WithResourceProvider adds a resource provider to the app
//
// newResourceProvider must be a function that returns a fxctx.ResourceProvider
// it can also take in any dependencies that you want to inject
// into the resource provider, that will be resolved by the fx framework
func (f *Builder) WithResourceProvider(newResourceProvider ...any) *Builder {
	for i := range newResourceProvider {
		f.options = append(f.options, fx.Provide(fxctx.AsResourceProvider(newResourceProvider[i])))
	}
	return f
}

// WithPrompt adds a prompt to the app
//
// newPrompt must be a function that returns a fxctx.Prompt
// it can also take in any dependencies that you want to inject
// into the prompt, that will be resolved by the fx framework
func (f *Builder) WithPrompt(newPrompt ...any) *Builder {
	for i := range newPrompt {
		f.options = append(f.options, fx.Provide(fxctx.AsPrompt(newPrompt[i])))
	}
	return f
}

// WithStdioTransport sets up the server to use stdio transport
//
// options can be used to configure the stdio transport
func (f *Builder) WithTransport(transport server.Transport) *Builder {
	f.transport = transport
	return f
}

// WithFxOptions adds additional fx.Options to fx.App instance
func (f *Builder) WithFxOptions(opts ...fx.Option) *Builder {
	f.options = append(f.options, fx.Options(opts...))
	return f
}

// WithName sets the name of the server
//
// The name would be returned to client during the initialization
// phase, if not set, it would default to "my-foxy-contexts-server"
func (f *Builder) WithName(name string) *Builder {
	f.implementation.Name = name
	return f
}

// WithVersion sets the version of the server
//
// The version would be returned to client during the initialization
// phase, if not set, it would default to "0.0.1"
func (f *Builder) WithVersion(version string) *Builder {
	f.implementation.Version = version
	return f
}

func (f *Builder) WithExtraServerOptions(extraOptions ...server.ServerOption) *Builder {
	f.extraServerOptions = append(f.extraServerOptions, extraOptions...)
	return f
}

// BuildFxApp builds the fx.App instance as configured by `With*` methods
func (f *Builder) BuildFxApp() (*fx.App, error) {
	if f.transport == nil {
		return nil, ErrNoTransportSpecified
	}

	f.options = append(f.options, fxctx.ProvideToolMux())
	f.options = append(f.options, fxctx.ProvideResourceMux())
	f.options = append(f.options, fxctx.ProvidePromptMux())
	f.options = append(f.options, fxctx.ProvideCompleteMux())
	f.options = append(f.options, fx.Provide(func() *session.SessionManager {
		return f.transport.GetSessionManager()
	}))
	f.options = append(f.options, f.provideServerLifecycle(f.transport))

	return fx.New(fx.Options(f.options...)), nil
}

// Run builds and runs the fx.App instance as configured by `With*` methods
func (f *Builder) Run() error {
	app, err := f.BuildFxApp()
	if err != nil {
		return err
	}
	app.Run()
	if f.transportError != nil {
		return f.transportError
	}
	return app.Err()
}

func (f *Builder) Err() error {
	return f.transportError
}

type ServerLifecycleParams struct {
	fx.In

	ToolMux     fxctx.ToolMux     `optional:"true"`
	ResourceMux fxctx.ResourceMux `optional:"true"`
	PromptMux   fxctx.PromptMux   `optional:"true"`
	CompleteMux fxctx.CompleteMux `optional:"true"`

	SessionManager *session.SessionManager
}

func (f *Builder) getServerCapabilities() *mcp.ServerCapabilities {
	if f.capabilities != nil {
		return f.capabilities
	}
	serverCapabilities := &mcp.ServerCapabilities{}
	return serverCapabilities
}

func (f *Builder) provideServerLifecycle(transport server.Transport) fx.Option {
	return fx.Invoke((func(
		lc fx.Lifecycle,
		p ServerLifecycleParams,
		shutdowner fx.Shutdowner,
	) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				serverStartOption := server.ServerStartCallbackOption{
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
				}
				options := append(f.extraServerOptions, serverStartOption)
				go func() {
					err := transport.Run(
						f.getServerCapabilities(),
						f.implementation,
						options...,
					)
					if err != nil {
						f.transportError = err
					}

					// shutdown the server when transport is done
					_ = shutdowner.Shutdown()
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return transport.Shutdown(ctx)
			},
		})
	}))
}
