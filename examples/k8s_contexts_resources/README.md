# K8s Contexts Resources Example

This example demonstrates how MCP resource could be defined using Foxy Contexts library.

Further assumes that you have installed Golang and (only for inspector) - Node.js.

## Trying with inspector 

To try this example with inspector, run the following command:

```bash
npx @modelcontextprotocol/inspector go run main.go
```

Then in browser open http://localhost:5173
, then click Connect
, then click List Resources
, you should see list of contexts and could click on them to see details

## Trying with Claude

Firstly you would need to run in this folder:

```bash
go install
```

Now in your Claude Desktop configuration file add the following:

```json
{
    "mcpServers": {
        "list_k8s_resources_example": {
            "command": "k8s_contexts_resources",
            "args": []
        }
    }
}
```

Now when you run Claude Desktop, you should see attachment button that would allow you to use k8s context as message attachment.

