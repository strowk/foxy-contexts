# Prompts

> Prompts enable servers to define reusable prompt templates and workflows that clients can easily surface to users and LLMs

MCP [prompts](https://modelcontextprotocol.io/docs/concepts/prompts) is a concept of MCP servers. Server should list prompts when requested with method `prompts/list` and retrieve when requested with method `prompts/get`.

Foxy Contexts allows easy way to define a prompt and register it within fx DI container.

## NewPrompt

In order to create new prompt you shall use `fxctx.NewPrompt` function. It accepts prompt name, description and function that would be called when prompt is called.

```go
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

```

## Provide prompt to fx

Now in order to let MCP server know about this prompt you would need to provide it to fx when creating fx app like this:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsPrompt(NewGreatPrompt)),
    // and more ...
)
```

`fxctx.AsPrompt` is a helper function that wraps your prompt into fx DI container and adds a certain tags, that would be used later to find all prompts and register them in MCP server.

## PromptMux

Discovery and registration of prompts would be done by PromptMux, which you can simply provide to fx all like this:

```go
fx.New(
    // other registered stuff ...
    fxctx.ProvidePromptMux(),
    // and more ...
)
```

To sum up you would need to provide all your prompts and PromptMux to fx:

```go
fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsPrompt(NewGreatPrompt)),
    fx.Provide(fxctx.AsPrompt(NewAnotherPrompt)),
    fxctx.ProvidePromptMux(),
    // and more ...
)

```

## Wrapping it all together

Finally, when starting the server, you would need to register handlers for prompts in your server using `server.ServerStartCallbackOption`:

```go

fx.New(
    // other registered stuff ...
    fx.Provide(fxctx.AsPrompt(NewGreatPrompt)),
    fx.Provide(fxctx.AsPrompt(NewAnotherPrompt)),
    fxctx.ProvidePromptMux(), // this makes sure that PromptMux can get all our prompts and is provided itself
    // and more ...

	// Start the server using stdio transport
    fx.Invoke((func(
        lc fx.Lifecycle,
        promptMux fxctx.PromptMux,
    ) {
        transport := stdio.NewTransport()
        lc.Append(fx.Hook{
            OnStart: func(ctx context.Context) error {
                go func() {
                    transport.Run(
                        &mcp.ServerCapabilities{
                            Prompts: &mcp.ServerCapabilitiesPrompts{
                                ListChanged: Ptr(false),
                            },
                        },
                        &mcp.Implementation{
                            Name:    "my-mcp-great-server",
                            Version: "0.0.1",
                        },
                        server.ServerStartCallbackOption{
                            Callback: func(s server.Server) {
                                // This makes sure that server is aware of the prompts
                                // we have registered and both can list and get them
                                promptMux.RegisterHandlers(s)
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

Check out complete example of MCP Server with prompt:

- [list_k8s_namespaces_prompt](https://github.com/strowk/foxy-contexts/tree/main/examples/list_k8s_namespaces_prompt) - provides a prompt listing k8s namespaces



