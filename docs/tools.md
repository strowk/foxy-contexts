# Tools

> Through tools, LLMs can interact with external systems, perform computations, and take actions in the real world.

MCP [tools](https://modelcontextprotocol.io/docs/concepts/tools) is a concept of MCP servers. Server should list tools when requested with method `tools/list` and call when requested with method `tools/call`.

Foxy Contexts allows easy way to define a tool and register it within fx DI container.

## NewTool

In order to create new tool you shall use `fxctx.NewTool` function. It accepts tool name, description and function that would be called when tool is called.

```go
func NewGreatTool() fxctx.Tool {
    return fxctx.NewTool(
        // This information about the tool would be used when it is listed:
        &mcp.Tool{
            Name: "my-great-tool",
            Description: Ptr("Lists files in the current directory"),
            InputSchema: mcp.ToolInputSchema{ // here we tell client what we expect as input
                Type:       "object",
                Properties: map[string]map[string]interface{}{},
                Required:   []string{},
            },
        }

        // This is the callback that would be executed when the tool is called:
        func(args map[string]interface{}) *mcp.CallToolResult {
            // here we can do anything we want
            return &mcp.CallToolResult{
                Content: []interface{}{
                    mcp.TextContent{
                        Type: "text",
                        Text: fmt.Sprintf("Hello, World!"),
                    },
                },
            }
        },
    )
}
```

## Provide tool to fx

Now in order to let MCP server know about this tool you would need to provide it to fx when creating fx app like this:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsTool(NewGreatTool)),
    // and more ...
)
```

`fxctx.AsTool` is a helper function that wraps your tool into fx DI container and adds a certain tags, that would be used later to find all tools and register them in MCP server.

## ToolMux

Discovery and registration of tools would be done by ToolsMux, which you can simply provide to fx all like this:

```go
fx.New(
    // other registered stuff ...
    fxctx.ProvideToolMux(),
    // and more ...
)
```

To sum up you would need to provide all your tools and ToolMux to fx:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsTool(NewGreatTool)),
    fx.Provide(fxctx.AsTool(NewAnotherTool)),
    fxctx.ProvideToolMux(),
    // and more ...
)

```

## Wrapping it all together

Finally, when starting the server, you would need to register handlers for tools in your server using `server.ServerStartCallbackOption`:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsTool(NewGreatTool)),
    fx.Provide(fxctx.AsTool(NewAnotherTool)),
    fxctx.ProvideToolMux(), // this makes sure that ToolMux can get all our tools and is provided itself
    // and more ...
    fx.Invoke((func(
        lc fx.Lifecycle,
        toolMux fxctx.ToolMux, // and here we take ToolMux from fx so we can register handlers for tools
    ) {
        transport := stdio.NewTransport()
        lc.Append(fx.Hook{
            OnStart: func(ctx context.Context) error {
                go func() {
                    transport.Run(
                        &mcp.ServerCapabilities{
                            Tools: &mcp.ServerCapabilitiesTools{
                                ListChanged: Ptr(false),
                            },
                        },
                        &mcp.Implementation{
                            Name:    "my-great-server",
                            Version: "0.0.1",
                        },
                        server.ServerStartCallbackOption{
                            Callback: func(s server.Server) {
                                // This makes sure that server is aware of the tools
                                // we have registered and both can list and call them
                                // serving for "tools/list" and "tools/call" requests
                                toolMux.RegisterHandlers(s)
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

Check out complete examples of MCP Servers with tools:

- [list_current_dir_files_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_current_dir_files_tool)
- [list_k8s_contexts_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_k8s_contexts_tool)