---
title: Resources
weight: 2
breadcrumbs: false
---

> Resources are a core primitive in the Model Context Protocol (MCP) that allow servers to expose data and content that can be read by clients and used as context for LLM interactions.

MCP [resources](https://modelcontextprotocol.io/docs/concepts/resources) is a concept of MCP servers. Server should list resources when requested with method `resources/list` and retrieve when requested with method `resources/read`.

Servers also can provide dynamic resources using templates by listing them via `resources/templates/list`, however this feature is not yet supported in Foxy Contexts.

In Foxy Contexts there are two ways to include resources in your server:

- using `fxctx.NewResource` to define static resources
- using `fxctx.NewResourceProvider` to define resource providers

Approach with resource provider is more flexible and allows to provide resources dynamically, however all such resources would be still included in response for `resources/list`, in contrasts to concept of templates (which are not yet supported).

## NewResource

In order to create new static resource you shall use `fxctx.NewResource` function.


```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/hello_world_resource/main.go" }
{{< snippet "examples/hello_world_resource/main.go:tool" "go" >}}
```

## Register resources and start server

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/hello_world_resource/main.go" }
{{< snippet "examples/hello_world_resource/main.go:server" "go" >}}
```

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

### Examples

Check out complete examples of MCP Servers with resources:

- [k8s_contexts_resources](https://github.com/strowk/foxy-contexts/tree/main/examples/k8s_contexts_resources) - provides k8s contexts as resources
- [hello_world_resource](https://github.com/strowk/foxy-contexts/tree/main/examples/hello_world_resource) - provides one static very simple resource
- [git_repository_resource](https://github.com/strowk/foxy-contexts/tree/main/examples/hello_world_resource) - provides one static resource with information about git repository
