# Foxy Contexts - Build MCP Servers Declaratively in Golang

Foxy contexts is a Golang library for building context servers supporting [Model Context Protocol](https://modelcontextprotocol.io/).

This library only supports server side of the protocol. Using it you can build context servers using declarative approach, by defining [tools](https://modelcontextprotocol.io/docs/concepts/tools), [resources](https://modelcontextprotocol.io/docs/concepts/resources) and [prompts](https://modelcontextprotocol.io/docs/concepts/prompts) and then registering them with your FoxyApp, which is using [uber's fx](https://github.com/uber-go/fx) under the hood and you can inject into its container as well.

With this approach you can easily colocate call/read/get logic and definitions of your tools/resources/prompts in a way that every tool/resource/prompt is placed in a separate place, but related code is colocated.

Here is list of features that are already implemented:

- [x] Base (lifecycle/ping)
- [x] Stdio Transport
- [x] SSE Transport
- [x] Tools
- [x] Package toolinput helps define tools input schema and validate arriving input
- [x] Resources - static
- [x] Resources - dynamic via Resource Providers
- [x] Prompts
- [x] Prompts Completion
- [x] Functional Testing package foxytest
- [x] Simple building of your MCP server with the power of Dependency Injection

And these are planned:

- [ ] Progress (planned)
- [ ] Resources - dynamic via Resource Templates (planned)
- [ ] Resource Templates completion (planned)
- [ ] Resource subscriptions
- [ ] Logging via MCP (planned)
- [ ] Sampling (planned)
- [ ] Roots (planned)
- [ ] Pagination (planned)
- [ ] Notifications list_changed (planned)

Check [docs](https://foxy-contexts.str4.io/) and [examples](https://github.com/strowk/foxy-contexts/tree/main/examples) to know more.

## Tool Example

For example try following

```bash
git clone https://github.com/strowk/foxy-contexts
cd foxy-contexts/examples/list_current_dir_files_tool
npx @modelcontextprotocol/inspector go run main.go
```
, then once inspector is started in browser open http://localhost:5173 and try to use list-current-dir-files.

Here's the code of that example from [examples/list_current_dir_files_tool/main.go](https://github.com/strowk/foxy-contexts/blob/main/examples/list_current_dir_files_tool/main.go) (in real world application you would probably want to split it into multiple files):


```go
package main

import (
	"fmt"
	"os"

	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/stdio"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// This example defines list-current-dir-files tool for MCP server, that prints files in the current directory
// , run it with:
// npx @modelcontextprotocol/inspector go run main.go
// , then in browser open http://localhost:5173
// , then click Connect
// , then click List Tools
// , then click list-current-dir-files

// NewListCurrentDirFilesTool defines a tool that lists files in the current directory
func NewListCurrentDirFilesTool() fxctx.Tool {
	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "list-current-dir-files",
			Description: Ptr("Lists files in the current directory"),
			InputSchema: mcp.ToolInputSchema{
				Type:       "object",
				Properties: map[string]map[string]interface{}{},
				Required:   []string{},
			},
		},

		// This is the callback that would be executed when the tool is called:
		func(args map[string]interface{}) *mcp.CallToolResult {
			files, err := os.ReadDir(".")
			if err != nil {
				return &mcp.CallToolResult{
					IsError: Ptr(true),
					Meta:    map[string]interface{}{},
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("failed to read dir: %v", err),
						},
					},
				}
			}
			var contents []interface{} = make([]interface{}, len(files))
			for i, f := range files {
				contents[i] = mcp.TextContent{
					Type: "text",
					Text: f.Name(),
				}
			}

			return &mcp.CallToolResult{
				Meta:    map[string]interface{}{},
				Content: contents,
				IsError: Ptr(false),
			}
		},
	)
}

func main() {
	app.
		NewFoxyApp().
		// adding the tool to the app
		WithTool(NewListCurrentDirFilesTool).
		// setting up server
		WithName("list-current-dir-files").
		WithVersion("0.0.1").
		WithTransport(stdio.NewTransport()).
		// Configuring fx logging to only show errors
		WithFxOptions(fx.Provide(func() *zap.Logger {
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
		).
		Run()
}

func Ptr[T any](v T) *T {
	return &v
}

```

