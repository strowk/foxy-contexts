# Streamable HTTP Example

This is a simple example of how to use the streamable HTTP transport with the MCP server protected by Outh2.

It implements both OAuth2 protected MCP server, authorization server as well as a client for another remote OAuth2 server, so that you can chain your authentication for example when your remote API also uses OAuth2, but it might not be OAuth2.1 compliant.

You will need ports 8080 and 8081 to be free for this example to work.

To start server, run this command:

```bash
go run main.go
```

Then try sending POST to see how server responds:

```bash
curl -X POST -i \
  -H "Content-Type: application/json" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' \
  http://localhost:8080/mcp
```

You should see output like this:

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json
Www-Authenticate: Bearer resource_metadata="/.well-known/oauth-protected-resource"
Date: Wed, 19 Nov 2025 09:06:55 GMT
Content-Length: 31

authorization header is missing
```

Start dex server for testing auth:

```bash
docker-compose up
```

Now start client:

```bash
cd example_client
go run main.go
```

This will print URL to visit for authentication. Open it in browser, login with "admin@example.com" / "testpwd", then click Grant Access.

After this, you should see that your client has made two requests to MCP server, one with auth and one without:

```
Token Response: {"access_token":"MJK1YTMWYWYTNDNIZS0ZMTAXLTK0NWUTYTAYOGIWYTU3M2FH","expires_in":7200,"refresh_token":"N2NIYTUYMZATYJIZYY01OTFJLWFHNTATMTAZNDZINDK3OWI1","token_type":"Bearer"}

MCP Ping Response: {"jsonrpc":"2.0","result":{},"id":0}

MCP Ping Response without auth: Unauthorized
```

You can then grab the access token from the output and use it to make authorized requests, like this:

```bash
token=ZWY1YJEXOGUTYJHIMS0ZOGFKLTHINJETNGMWNTVHNZHLMWRJ

curl -X POST -i \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' \
  http://localhost:8080/mcp
```

Note that whenever you do it, your server would print "Using remote token from session: <token>" in the console, showing that tool can access the remote token via session.

In this example remote token is not actually used to make any requests, but you could easily extend the tool to do so.

## Using session

Try making such request to call the tool:

```bash
curl -X POST -i \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":0}' \
  http://localhost:8080/mcp
```

You should see output like this:

```
Content-Type: application/json
Mcp-Session-Id: 7fd56bef-e4a9-43c5-8c9e-77ea4a588b77
Date: Sun, 16 Nov 2025 04:12:53 GMT
Content-Length: 106

{"jsonrpc":"2.0","result":{"content":[{"text":"Sup, saving greatness to session","type":"text"}]},"id":0}
```

Grab session id from the response header and use it in the next request:

```bash
curl -X POST -i \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: 7fd56bef-e4a9-43c5-8c9e-77ea4a588b77" \
  -H "Authorization: Bearer $token" \
  -d '{"method":"tools/call", "params": {"name": "my-great-tool", "arguments": {}},"id":1}' \
  http://localhost:8080/mcp
```

This should print out different body:

```json
{"jsonrpc":"2.0","result":{"content":[{"text":"Sup, already great","type":"text"}]},"id":1}
```

The result is different for this session, but if you send another request without session id, it would give you the same result as before. 
This is because of how tool keeps session state using SessionManager.
