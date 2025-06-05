# List Current Directory Files Tool

This example demonstrates how MCP resource could be defined using Foxy Contexts library.

Further assumes that you have installed Golang and (only for inspector) - Node.js.

## Trying with inspector

To try this example with inspector, run the following command:

```bash
npx @modelcontextprotocol/inspector go run main.go
```

Then in browser open http://localhost:6274
then click Connect
then click List Tools
then click list-current-dir-files

You should see list of files in current directory.

## Trying with Claude

Firstly you would need to run in this folder:

```bash
go install
```

, and make sure that you have configured golang bin directory in your PATH.

Now in your Claude Desktop configuration file add the following:

```json
{
    "mcpServers": {
        "list_current_dir_files_tool": {
            "command": "list_current_dir_files_tool",
            "args": []
        }
    }
}
```

Now when you run Claude Desktop, you can ask Claude to list files in current directory.
It is probably not very useful, as you would see files in whatever directory Claude is running from.


## Auto test

This will run integration test:

```bash
go test
```

Check out testcase in [testdata/list_and_call_test.yaml](testdata/list_and_call_test.yaml) and test runner setup in [main_test.go](main_test.go).