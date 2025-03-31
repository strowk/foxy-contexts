# Streamable HTTP Example

This is a simple example of how to use the streamable HTTP transport with the MCP server.

You will need port 8080 to be free for this example to work.

To start server, run this command:

```bash
go run main.go
```

Then try sending POST to see how server responds:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' \
  http://localhost:8080/mcp
```

You can also autotest this example like this:

```bash
go test
```

, or with mcp-autotest:

```bash
mcp-autotest run -u http://localhost:8080/mcp testdata -- go run main.go
```


## Using session

Try making such request to call the tool:

```bash
curl -X POST -i \
  -H "Content-Type: application/json" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' \
  http://localhost:8080/mcp
```

You should see output like this:

```
Content-Type: application/json
Mcp-Session-Id: 25b4dda4-90dc-4616-a298-24a8a23e90ad
Date: Mon, 31 Mar 2025 22:28:51 GMT
Content-Length: 106

{"jsonrpc":"2.0","result":{"content":[{"text":"Sup, saving greatness to session","type":"text"}]},"id":0}
```

Grab session id from the response header and use it in the next request:

```bash
curl -X POST -i \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: 25b4dda4-90dc-4616-a298-24a8a23e90ad" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":1}' \
  http://localhost:8080/mcp
```

This should print out different body:

```json
{"jsonrpc":"2.0","result":{"content":[{"text":"Sup, already great","type":"text"}]},"id":1}
```

The result is different for this session, but if you send another request without session id, it would give you the same result as before. 
This is because of how tool keeps session state using SessionManager.