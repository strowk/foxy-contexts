# Resources

> Resources are a core primitive in the Model Context Protocol (MCP) that allow servers to expose data and content that can be read by clients and used as context for LLM interactions.

MCP [resources](https://modelcontextprotocol.io/docs/concepts/resources) is a concept of MCP servers. Server should list resources when requested with method `resources/list` and retrieve when requested with method `resources/read`.

Servers also can provide dynamic resources using templates by listing them via `resources/templates/list`, however this feature is not yet supported in Foxy Contexts.

In Foxy Contexts there are two ways to include resources in your server:

- using `fxctx.NewResource` and `fxctx.AsResource` to define and provide resources
- using `fxctx.NewResourceProvider` and `fxctx.AsResourceProvider` to define and provide resource providers

Approach with resource provider is more flexible and allows to provide resources dynamically, however all such resources would be still included in response for `resources/list`, in contrasts to concept of templates (which are not yet supported).

## NewResource

In order to create new static resource you shall use `fxctx.NewResource` function.

```go
func NewHelloWorldResource() fxctx.Resource {
    return fxctx.NewResource(
        mcp.Resource{
            Name:        "hello-world",
            Uri:         "hello-world://hello-world",
            MimeType:    Ptr("application/json"),
            Description: Ptr("Hello World Resource"),
            Annotations: &mcp.ResourceAnnotations{
                Audience: []mcp.Role{
                    mcp.RoleAssistant, mcp.RoleUser,
                },
            },
        },
        func(uri string) (*mcp.ReadResourceResult, error) {
            return &mcp.ReadResourceResult{
                Contents: []interface{}{
                    mcp.TextResourceContents{
                        MimeType: Ptr("application/json"),
                        Text:     `{"hello": "world"}`,
                        Uri:      uri,
                    },
                },
            }, nil
        },
    )
},

```

## Provide resource to fx

Now in order to let MCP server know about this resource you would need to provide it to fx when creating fx app like this:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsResource(NewHelloWorldResource)),
    // and more ...
)
```

`fxctx.AsResource` is a helper function that wraps your resource into fx DI container and adds a certain tags, that would be used later to find all resources and register them in MCP server.

Head further to [ResourceMux](#ResourceMux) to see how to register such resource in MCP server, or see next section to learn how to provide dynamic resources.

## NewResourceProvider

In order to create new resource provider that would be returning resources dynamically, you shall use `fxctx.NewResourceProvider` function. It would then take two functions - one in order to list resources and another to read them.

```go
func NewGreatResourceProvider() fxctx.ResourceProvider {
    return fxctx.NewResourceProvider(
        // This is the callback that would be executed when the resources/list is requested:
        func() ([]mcp.Resource, error) {
            return []mcp.Resource{
                {
                    Name: "my-great-resource-one",
                    Description: Ptr("Does something great"),
                    Uri: "/resources/great/one",
                },
            }, nil
        },
        //  This function reads the resource for a given uri to run when resources/read is requested:
		func(uri string) (*mcp.ReadResourceResult, error) {

            // you would probably be doing something more complicated here
            // like reading from a database or calling an external service
            // based on what you have parsed from the uri
            if uri == "/resources/great/one" {
                return &mcp.ReadResourceResult{
                    Contents: []interface{}{
                        mcp.TextResourceContents{
                            MimeType: Ptr("application/json"),
                            Text:     string(`{"great": "resource"}`),
                            Uri:      uri,
                        }
                    },
                }, nil
            }

            // this error would be wrapped in JSON-RPC error response
            return nil, fmt.Errorf("resource not found")
        },
    )
}
```

## Provide resource provider to fx

Now in order to let MCP server know about this resource provider you would need to provide it to fx when creating fx app like this:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsResourceProvider(NewGreatResourceProvider)),
    // and more ...
)
```

`fxctx.AsResourceProvider` is a helper function that wraps your resource provider into fx DI container and adds a certain tags, that would be used later to find all resource providers and register them in MCP server.

## ResourceMux

Discovery and registration of resources would be done by ResourceMux, which you can simply provide to fx all like this:

```go
fx.New(
    // other registered stuff ...
    fxctx.ProvideResourceMux(),
    // and more ...
)
```

To sum up you would need to provide all your resource providers and ResourceMux to fx:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsResourceProvider(NewGreatResourceProvider)), // dynamic resources are provided by resource providers
    fx.Provide(fxctx.AsResourceProvider(NewAnotherResourceProvider)), // you can provide multiple resource providers
    fx.Provide(fxctx.AsResource(NewHelloWorldResource)), // static resources are provided directly
    fxctx.ProvideResourceMux(),
    // and more ...
)
```

## Wrapping it all together

Finally, when starting the server, you would need to register handlers for resources in your server using `server.ServerStartCallbackOption`:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsResourceProvider(NewGreatResourceProvider)),
    fx.Provide(fxctx.AsResourceProvider(NewAnotherResourceProvider)),
    fx.Provide(fxctx.AsResource(NewHelloWorldResource)),
    fxctx.ProvideResourceMux(), // this makes sure that ResourceMux can get all our resource providers and is provided itself

    // Start the server using stdio transport
    fx.Invoke((func(
        lc fx.Lifecycle,
        resourceMux fxctx.ResourceMux,
    ) {
        transport := stdio.NewTransport()
        lc.Append(fx.Hook{
            OnStart: func(ctx context.Context) error {
                go func() {
                    transport.Run(
                        &mcp.ServerCapabilities{
                            Resources: &mcp.ServerCapabilitiesResources{
                                ListChanged: Ptr(false),
                                Subscribe:   Ptr(false),
                            },
                        },
                        &mcp.Implementation{
                            Name:    "my-mcp-great-server",
                            Version: "0.0.1",
                        },
                        server.ServerStartCallbackOption{
                            Callback: func(s server.Server) {
                                // This makes sure that server is aware of the tools
                                // we have registered and both can list and call them
                                resourceMux.RegisterHandlers(s)
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
)
```

### Examples

Check out complete examples of MCP Servers with resources:

- [k8s_contexts_resources](https://github.com/strowk/foxy-contexts/tree/main/examples/k8s_contexts_resources) - provides k8s contexts as resources
- [hello_world_resource](https://github.com/strowk/foxy-contexts/tree/main/examples/hello_world_resource) - provides one static very simple resource
- [git_repository_resource]https://github.com/strowk/foxy-contexts/tree/main/examples/hello_world_resource) - provides one static resource with information about git repository
