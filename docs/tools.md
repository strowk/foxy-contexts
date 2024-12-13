# Tools

> Through tools, LLMs can interact with external systems, perform computations, and take actions in the real world.

MCP [tools](https://modelcontextprotocol.io/docs/concepts/tools) is a concept of MCP servers. Server should list tools when requested with method `tools/list` and call when requested with method `tools/call`.

Foxy Contexts allows easy way to define a tool and register it within fx DI container.

## NewTool

In order to create new tool you shall use `fxctx.NewTool` function. It accepts tool name, description and function that would be called when tool is called.

```go
--8<-- "examples/simple_great_tool/main.go:tool"
```

## Register tool and start server

```go
--8<-- "examples/simple_great_tool/main.go:server"
```

### Examples

Check out complete examples of MCP Servers with tools:

- [list_current_dir_files_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_current_dir_files_tool)
- [list_k8s_contexts_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_k8s_contexts_tool)