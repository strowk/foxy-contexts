package app

import (
	"context"
	"errors"

	"github.com/strowk/foxy-contexts/pkg/auth"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/session"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
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

	logger *zap.Logger
}

func (f *Builder) WithAuthorization(authorization auth.Authorization) *Builder {
	f.options = append(f.options, fx.Provide(func() auth.Authorization { return authorization }))
	return f
}

// WithTool adds a tool to the app
//
// newTool must be a function that returns a fxctx.Tool
// it can also take in any dependencies that you want to inject
// into the tool, that will be resolved by the fx framework
func (f *Builder) WithTool(newTool any) *Builder {
	f.options = append(f.options, fx.Provide(fxctx.AsTool(newTool)))
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
func (f *Builder) WithResource(newResource any) *Builder {
	f.options = append(f.options, fx.Provide(fxctx.AsResource(newResource)))
	return f
}

// WithResourceProvider adds a resource provider to the app
//
// newResourceProvider must be a function that returns a fxctx.ResourceProvider
// it can also take in any dependencies that you want to inject
// into the resource provider, that will be resolved by the fx framework
func (f *Builder) WithResourceProvider(newResourceProvider any) *Builder {
	f.options = append(f.options, fx.Provide(fxctx.AsResourceProvider(newResourceProvider)))
	return f
}

// WithPrompt adds a prompt to the app
//
// newPrompt must be a function that returns a fxctx.Prompt
// it can also take in any dependencies that you want to inject
// into the prompt, that will be resolved by the fx framework
func (f *Builder) WithPrompt(newPrompt any) *Builder {
	f.options = append(f.options, fx.Provide(fxctx.AsPrompt(newPrompt)))
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

func (f *Builder) WithLogger(logger *zap.Logger) *Builder {
	f.logger = logger
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
	f.options = append(f.options, f.provideServerLifecycle())

	if f.logger == nil {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zap.ErrorLevel)
		logger, _ := cfg.Build()
		f.logger = logger
	}

	fxEventLogger := fx.WithLogger(
		func() fxevent.Logger {
			return &fxevent.ZapLogger{Logger: f.logger}
		},
	)

	return fx.New(
		fxEventLogger,
		fx.Module("transport",
			fx.Provide(f.getTransportForFx),
		),
		fx.Module(
			"mcp-server",
			fx.Invoke(func(_ server.Transport) {
				// this is a no-op, but it ensures that the transport is provided
				// before the rest of application is prepared in order to avoid
				// deadlocks when tools or their dependencies are blocking
				// transport shutdown, like for example http server in streamable http
				// transport would be blocked on waiting for all running requests to finish
			}),
			fx.Options(f.options...),
		),
	), nil
}

func (f *Builder) getTransportForFx(lc fx.Lifecycle) server.Transport {
	lc.Append(fx.StopHook(f.transport.Shutdown))
	return f.transport
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

	Authorization auth.Authorization `optional:"true"`
}

func (f *Builder) getServerCapabilities() *mcp.ServerCapabilities {
	if f.capabilities != nil {
		return f.capabilities
	}
	serverCapabilities := &mcp.ServerCapabilities{}
	return serverCapabilities
}

func (f *Builder) provideServerLifecycle() fx.Option {
	return fx.Invoke((func(
		lc fx.Lifecycle,
		params ServerLifecycleParams,
		shutdowner fx.Shutdowner,
		transport server.Transport,
	) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if params.Authorization != nil {
					err := plugAuthorizationInTransport(transport, params.Authorization)
					if err != nil {
						return err
					}
				}
				serverStartOption := server.ServerStartCallbackOption{
					Callback: func(s server.Server) {
						if params.ToolMux != nil {
							params.ToolMux.RegisterHandlers(s)
						}
						if params.ResourceMux != nil {
							params.ResourceMux.RegisterHandlers(s)
						}
						if params.PromptMux != nil {
							params.PromptMux.RegisterHandlers(s)
						}
						if params.CompleteMux != nil {
							params.CompleteMux.RegisterHandlers(s)
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
		})
	}))
}

func plugAuthorizationInTransport(
	transport server.Transport,
	authorization auth.Authorization,
) error {
	authorizeableTransport, ok := transport.(auth.AuthorizeableTransport)
	if !ok {
		return auth.ErrInvalidTransport
	}
	return authorizeableTransport.PlugInAuthorization(authorization)
}
