# List K8s Contexts Tool

This example demonstrates how MCP tool could be defined using Foxy Contexts library.

Further assumes that you have installed Golang and (only for inspector) - Node.js.

The assumption is that you also have some k8s contexts configured on your machine.

## Trying with inspector

To try this example with inspector, run the following command:

```bash
npx @modelcontextprotocol/inspector go run main.go
```

Then in browser open http://localhost:6274
then click Connect
then click List Tools
then click list-k8s-contexts

You should see list of k8s contexts.


## Trying with Claude

Firstly you would need to run in this folder:

```bash
go install
```

Now in your Claude Desktop configuration file add the following:

```json
{
    "mcpServers": {
        "list_k8s_contexts_tool_example": {
            "command": "list_k8s_contexts_tool",
            "args": []
        }
    }
}
```

Now when you run Claude Desktop, you can ask Claude to list k8s contexts.