---
breadcrumbs: false
weight: 4
---

# app.Builder

app.Builder is a builder around uber's fx that combines fx DI capabilities with foxy-contexts MCP server implementation and gives simple interface to combine your MCP primitives into a single standalone application.

## Usage

Simple example of wrapping one tool into an application, then running it with stdio transport:

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_great_tool/main.go" }
{{< snippet "examples/simple_great_tool/main.go:server" "go" >}}
```


### Providing additional server options

Normally app.Builder preconfigure server for you, but you can provide additional server options to the application by using `WithExtraServerOptions` option.

Say, for example this would make server to turn on logging to stderr:

```go
WithExtraServerOptions(server.LoggerOption{
    Logger: foxyevent.NewSlogLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))).WithLogLevel(slog.LevelInfo),
})
```