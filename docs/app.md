# FoxyApp

FoxyApp is a builder around uber's fx that combines fx DI capabilities with foxy-contexts MCP server implementation and gives simple interface to combine your MCP primitives into a single application.

## Usage

Simple example of wrapping one tool into an application, then running it with stdio transport:

```go
--8<-- "examples/simple_great_tool/main.go:server"
```

### Providing additional server options

Normally FoxyApp preconfigure server for you, but you can provide additional server options to the application by using `WithExtraServerOptions` option.

Say, for example this would make server to turn on logging to stderr:

```go
WithExtraServerOptions(server.LoggerOption{
    Logger: foxyevent.NewSlogLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))).WithLogLevel(slog.LevelInfo),
})
```