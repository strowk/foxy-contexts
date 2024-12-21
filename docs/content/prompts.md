---
breadcrumbs: false
weight: 3
---

# Prompts

> Prompts enable servers to define reusable prompt templates and workflows that clients can easily surface to users and LLMs

MCP [prompts](https://modelcontextprotocol.io/docs/concepts/prompts) is a concept of MCP servers. Server should list prompts when requested with method `prompts/list` and retrieve when requested with method `prompts/get`.

Foxy Contexts allows easy way to define a prompt and register it within fx DI container.

## NewPrompt

In order to create new prompt you shall use `fxctx.NewPrompt` function. It accepts prompt name, description and function that would be called when prompt is called.

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_prompt/main.go" }
{{< snippet "examples/simple_prompt/main.go:prompt" "go" >}}
```

## Register prompt and start server

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/simple_prompt/main.go" }
{{< snippet "examples/simple_prompt/main.go:server" "go" >}}
```

### Examples

Check out complete example of MCP Server with prompt:

- [simple_prompt](https://github.com/strowk/foxy-contexts/tree/main/examples/simple_prompt) - provides a very basic prompt
- [list_k8s_namespaces_prompt](https://github.com/strowk/foxy-contexts/tree/main/examples/list_k8s_namespaces_prompt) - provides a prompt listing k8s namespaces
