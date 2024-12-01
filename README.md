# Foxy Contexts

Foxy contexts is a library for building context servers supporting [Model Context Protocol](https://modelcontextprotocol.io/).

This library only supports server side of the protocol. Using it you can build context servers using declarative approach, by defining [tools](https://modelcontextprotocol.io/docs/concepts/tools), [resources](https://modelcontextprotocol.io/docs/concepts/resources) and [prompts](https://modelcontextprotocol.io/docs/concepts/prompts) and then registering them within DI using [uber's fx](https://modelcontextprotocol.io/docs/concepts/resources).

## Try it out

Check examples in [examples](./examples) directory.

Fox example:

```bash
cd examples/list_k8s_contexts_tool
npx @modelcontextprotocol/inspector go run main.go
```
, then in browser open http://localhost:5173 and try to use list-k8s-contexts tool, then check out implementation in [examples/list_k8s_contexts_tool/main.go](./examples/list_k8s_contexts_tool/main.go).




