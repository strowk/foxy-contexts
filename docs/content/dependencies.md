---
breadcrumbs: false
weight: 5
---

# Dependencies

The power of Foxy Contexts comes from being based on DI concept facilitated by [fx](https://github.com/uber-go/fx) library. This allows to easily extract common parts of your server into separate packages and reuse them in different tools, prompts and other parts of your server.

You can learn more about fx in [official documentation](https://pkg.go.dev/go.uber.org/fx), here there are just some common patterns that you might find useful.

## Extracting dependency

When you have identified that there is some bit that you need to repeat across tools, and maybe also for performance reasons you want to extract it and share, you can start with creating a function that creates that shared part. 

Following is example for extracting k8s configuration:

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/list_k8s_contexts_tool/k8s.go" }
{{< snippet "examples/list_k8s_contexts_tool/k8s.go:dependency_create" "go" >}}
```

With this in place, you can then define how the dependency would be injected into the tool you are creating:

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/list_k8s_contexts_tool/main.go" }
{{< snippet "examples/list_k8s_contexts_tool/main.go:dependency_inject" "go" >}}
```

The final step would be to provide everything to your `app.Builder`:

```go { filename_uri_base="https://github.com/strowk/foxy-contexts/blob/main" filename="examples/list_k8s_contexts_tool/main.go" }
{{< snippet "examples/list_k8s_contexts_tool/main.go:dependency_provide" "go" >}}
```

