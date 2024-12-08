# Testing

## Integration Testing with foxytest using MCP Story format

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

Tests could be single or multi-document YAML files. Each document should be a valid MCP Story, that means it should have `case` property with name of the test case and any number of properties prefixed with `in` and `out`, which would represent JSON-RPC 2.0 messages sent from client to server (`in`) and from server to client (`out`).

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

See following examples with tests:
- [examples/git_repository_resource/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/git_repository_resource/main_test.go)
- [examples/hello_world_resource/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/hello_world_resource/main_test.go)
- [examples/k8s_contexts_resources/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/k8s_contexts_resources/main_test.go)
- [examples/list_current_dir_files_tool/main_test.go](https://github.com/strowk/foxy-contexts/blob/main/examples/list_current_dir_files_tool/main_test.go)
