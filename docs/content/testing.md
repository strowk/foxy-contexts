---
breadcrumbs: false
weight: 5
---

# Testing

## foxytest

In order to test your server, package `foxytest` is provided that allows you to easily start your server and test it using pre-defined JSON-RPC 2.0 messages.

`foxytest` currently only supports stdio transport.

Here is an example how to setup integration tests:

```go
import (
	"testing"

	"github.com/strowk/foxy-contexts/pkg/foxytest"
)

func TestWithFoxytest(t *testing.T) {
	ts, err := foxytest.Read("testdata")
	if err != nil {
		t.Fatal(err)
	}
	ts.WithExecutable("go", []string{"run", "main.go"})
	ts.WithLogging() // this adds logging to the test runner, you could see it if you run tests with -v flag
	cntrl := foxytest.NewTestRunner(t)
	ts.Run(cntrl)
	ts.AssertNoErrors(cntrl)
}
```

In folder `testdata`, you should have files with names ending on `_test.yaml`. These files should contain MCP Stories that would describe the test scenario. For example:

```yaml
case: Empty tools list
in_list_tools: {"jsonrpc":"2.0","method":"tools/list","id":1}
out_no_tools: {"jsonrpc":"2.0","result":{"tools":[]},"id":1}
```

This says that when client sends `tools/list` request, server should respond with empty list of tools.

Tests could be single or multi-document YAML files. Each document should be in a valid MCP Cases format, that means it can have `case` property with name of the test case and any number of properties prefixed with `in` and `out`, which would represent JSON-RPC 2.0 messages sent from client to server (`in`) and from server to client (`out`).

When you run the test by executing `go test`, the test runner will start your server (by running `go run main.go` in this case), connect to stdio transport and send the message under `in` property. 

It will then wait for the response from server and compare it with the message under `out` property. If message is matching JSON structure, it will pass the test, otherwise it will fail and would print the diff between expected and actual JSON's.

If you need more information from test run, and have configured logging with `ts.WithLogging()` you can then use `verbose` flag when running the test: `go test -v`, you will then see the output like this:

```
=== RUN   TestWithFoxytest
    testsuite.go:55: setting up test suite
    testsuite.go:59: running command: go run main.go
    testsuite.go:59: running 1 tests
    testsuite.go:55: waiting for command to finish
    testsuite.go:59: expecting output: {"id":1,"jsonrpc":"2.0","result":{"tools":[]}}
    testsuite.go:59: sending input: {"id":1,"jsonrpc":"2.0","method":"tools/list"}
    testsuite.go:59: output matches: {"jsonrpc":"2.0","result":{"tools":[]},"id":1}
    testsuite.go:55: tests done
    testsuite.go:55: running after all
    testsuite.go:55: stop executable
    testsuite.go:55: finished reading output
    testsuite.go:55: executable stopped
--- PASS: TestWithFoxytest (1.39s)
```

When failed, package attempts to pretty print the diff between expected and actual JSON's, so you can quickly locate the issue.

Here is an example of real world failing test:

```
    testsuite.go:54:
          "result": {
            "content": {
              "0": {
                "text": {
                ^ value does not match the embedded regex:
        expected to match: "{"name":"k3d-mcp-k8s-test-server-0","status":"Active","age":"/[\\d][sm]/","createdAt":"/[\\d]{4}/-/[\\d]{2}/-/[\\d]{2}/T/[\\d]{2}/:/[\\d]{2}/:/[\\d]{2}/Z"}",
                  but got: "{"name":"k3d-mcp-k8s-integration-test-server-0","status":"Unknown","age":"1m0s"}"
```

Output is not a valid JSON, it just attempts to show the difference at the right point in structure.


## Match with regex

Sometimes when you have dynamic values in your responses, you might want to use regular expressions to match them. You can do this by using `!!re` tag in your expected output. For example:

```yaml
case: List tools
in_list_tools: {"jsonrpc":"2.0","method":"tools/list","id":1}
out_list_tools: {"jsonrpc":"2.0","result":{"tools":[ { "name": !!re "list-[a-z]+" } ]},"id":1}
```

This example is somewhat oversimplified, in reality you would probably use `!!re` with timestamps, UUIDs or other dynamic values, rather than tool names.
It does, however, demonstrate that strings with `!!re` tag would be treated as regular expressions and would be matched against actual values using regular expression matching instead of string equality.

## Embedded regex

Embedded regular expression is just a regular expression that is embedded in a string. It is used to match a part of the string. For example:

```yaml
case: List tools
in_list_tools: {"jsonrpc":"2.0","method":"tools/list","id":1}
out_list_tools: {"jsonrpc":"2.0","result":{"tools":[ { "name": !!ere "list-/[a-z]+/-tool" } ]},"id":1}
```

Essentially `!!ere` allows you to treat most of the string as just regular string, but have a part (or parts) of it treated as regular expression.
Everything inside slashes `/` would be treated as regular expression, here "list-" and "-tool" are regular strings, but `[a-z]+` is a regular expression that would match any lowercase letters non-empty string, so it would match "list-abc-tool", "list-xyz-tool" and so on.

This approach allows you not to think how to escape regular expression syntax in the rest of your string and only match bits that you need to be dynamic.

### Escaping forward slashes

You do, however need to escape slashes `/` in places of your string where you want to use them, but not designate regular expression, for example: `"url": !! "https:\\/\\/foxy-contexts.str4.io\\//[a-z]+/"` would match `"url": "https://foxy-contexts.str4.io/abc"`.
In here `\\/` is used to become `/` and `[a-z]+` is used to match any lowercase letters non-empty string. The reason why there are two backslashes `\\` is because in YAML strings backslash is an escape character, so to have a single backslash in the string you need to escape it with another backslash.

## Examples


See following examples with tests:

- [examples/git_repository_resource/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/git_repository_resource/main_test.go)
- [examples/hello_world_resource/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/hello_world_resource/main_test.go)
- [examples/k8s_contexts_resources/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/k8s_contexts_resources/main_test.go)
- [examples/list_current_dir_files_tool/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/list_current_dir_files_tool/main_test.go)
