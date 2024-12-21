---
title: Tools
weight: 1
breadcrumbs: false
---

> Through tools, LLMs can interact with external systems, perform computations, and take actions in the real world.

MCP [tools](https://modelcontextprotocol.io/docs/concepts/tools) is a concept of MCP servers. Server should list tools when requested with method `tools/list` and call when requested with method `tools/call`.

Foxy Contexts allows easy way to define a tool and register it within fx DI container.

## NewTool

In order to create new tool you shall use `fxctx.NewTool` function. It accepts tool name, description and function that would be called when tool is called.


```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_great_tool/main.go" }
{{< snippet "examples/simple_great_tool/main.go:tool" "go" >}}
```

## Register tool and start server

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_great_tool/main.go" }
{{< snippet "examples/simple_great_tool/main.go:server" "go" >}}
```

## Using toolinput package

In order to define input schema for your tool, you can use `toolinput` package. It allows you to define input schema and validate arriving input.

Here is an example of creating schema, giving it to the tool and validating input:

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_great_tool/main.go" }
{{< snippet "examples/list_k8s_contexts_tool/main.go:toolinput" "go" >}}
```

### Examples

Check out complete examples of MCP Servers with tools:

- [list_current_dir_files_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_current_dir_files_tool)
- [list_k8s_contexts_tool](https://github.com/strowk/foxy-contexts/tree/main/examples/list_k8s_contexts_tool)