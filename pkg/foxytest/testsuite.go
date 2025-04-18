package foxytest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type TestSystem interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

// TestRunner is an interface that is compatible with the testing.T interface
// , so that the test suite can be run with the standard go test runner
// or with any other runner that implements this interface, particularly
// it is used by `mst` command line runner allowing to use this
// test suite runner from languages other than Go
type TestRunner interface {
	TestSystem
	Run(name string, f func(t TestRunner)) bool
}

type StdRunner struct {
	t *testing.T
}

func NewTestRunner(t *testing.T) *StdRunner {
	return &StdRunner{
		t: t,
	}
}

func (tc *StdRunner) Fatal(args ...interface{}) {
	tc.t.Helper()
	tc.t.Fatal(args...)
}

func (tc *StdRunner) Fatalf(format string, args ...interface{}) {
	tc.t.Helper()
	tc.t.Fatalf(format, args...)
}

func (tc *StdRunner) Errorf(format string, args ...interface{}) {
	tc.t.Helper()
	tc.t.Errorf(format, args...)
}

func (tc *StdRunner) Log(args ...interface{}) {
	tc.t.Helper()
	tc.t.Log(args...)
}

func (tc *StdRunner) Logf(format string, args ...interface{}) {
	tc.t.Helper()
	tc.t.Logf(format, args...)
}

func (tc *StdRunner) Run(name string, f func(t TestRunner)) bool {
	tc.t.Run(name, func(t *testing.T) {
		f(NewTestRunner(t))
	})
	return true
}

// TestSuite represents a collection of tests
// that can be run together
type TestSuite interface {
	WithBeforeAll(func() error) TestSuite
	WithAfterAll(func() error) TestSuite
	WithBeforeEach(func() error) TestSuite
	WithAfterEach(func() error) TestSuite
	WithLogging() TestSuite

	WithExecutable(command string, args []string) TestSuite
	WithTransport(transport TestTransport) TestSuite

	Run(t TestRunner)
	Errors() []error
	AssertNoErrors(t TestRunner)
	GetTests() []Test
}

type testSuite struct {
	path string

	beforeAll  []func() error
	afterAll   []func() error
	beforeEach []func() error
	afterEach  []func() error
	tests      []Test

	inputChan  chan TestInput
	outputChan chan string

	// targetInput is to write to the target process, where the SuT takes input
	targetInput io.Writer

	// targetOutput is to read from the target process, where the SuT writes output
	targetOutput io.Reader

	command string
	args    []string

	errors []error

	logging bool

	executableDone chan struct{}
	testsDone      chan struct{}
	completed      chan struct{}

	transport TestTransport
}

func (ts *testSuite) GetTests() []Test {
	return ts.tests
}

func (ts *testSuite) WithBeforeAll(f func() error) TestSuite {
	ts.beforeAll = append(ts.beforeAll, f)
	return ts
}

func (ts *testSuite) WithAfterAll(f func() error) TestSuite {
	ts.afterAll = append(ts.afterAll, f)
	return ts
}

func (ts *testSuite) WithBeforeEach(f func() error) TestSuite {
	ts.beforeEach = append(ts.beforeEach, f)
	return ts
}

func (ts *testSuite) WithAfterEach(f func() error) TestSuite {
	ts.afterEach = append(ts.afterEach, f)
	return ts
}

func (ts *testSuite) WithExecutable(command string, args []string) TestSuite {
	ts.command = command
	ts.args = args
	return ts
}

func (ts *testSuite) WithLogging() TestSuite {
	ts.logging = true
	for _, test := range ts.tests {
		test.SetLogging(true)
	}
	return ts
}

func (ts *testSuite) WithTransport(transport TestTransport) TestSuite {
	ts.transport = transport
	return ts
}

func (ts *testSuite) AssertNoErrors(t TestRunner) {
	if len(ts.errors) > 0 {
		t.Errorf("errors: %v", ts.errors)
	}
}

func (ts *testSuite) Errors() []error {
	return ts.errors
}

func (ts *testSuite) runTests(t TestRunner) []error {
	if ts.transport == nil {
		ts.transport = &TestTransportStdio{}
	}
	// signal that tests are done at the end
	defer close(ts.testsDone)

	// firstly run all before all functions,
	// returning if any of them fails
	for _, f := range ts.beforeAll {
		err := f()
		if err != nil {
			ts.errors = append(ts.errors, err)
			return nil
		}
	}

	// setup the test suite - that would run the target process if necessary
	// and setup pipes for input and output
	if ts.logging {
		t.Log("setting up test suite")
	}
	ts.setup(t)

	if ts.logging {
		t.Logf("running %d tests", len(ts.tests))
	}
	// run all tests
	testErrors := []error{}
	for _, test := range ts.tests {
		test.Run(t)
	}

	if ts.logging {
		t.Log("tests done")
	}

	if len(testErrors) > 0 {
		ts.errors = append(ts.errors, testErrors...)
	}
	return testErrors
}

func (ts *testSuite) Run(t TestRunner) {
	ts.runTests(t)

	// run all after all functions
	// regardless of whether the tests failed or not
	if ts.logging {
		t.Log("running after all")
	}
	for _, f := range ts.afterAll {
		err := f()
		if err != nil {
			ts.errors = append(ts.errors, err)
		}
	}

	<-ts.completed
	if ts.logging {
		t.Logf("test suite %s completed", ts.path)
	}
}

type Test interface {
	Run(t TestRunner)
	SetLogging(bool)
	GetTestCases() []TestCase
}

type TestCase interface {
	GetName() string
	GetInputs() []TestInput
	GetOutputs() []map[string]any
}

type TestInput interface {
	getForMarshalling() any
}

type singleInput struct {
	input map[string]any
}

func (si *singleInput) getForMarshalling() any {
	return si.input
}

type batchInput struct {
	inputs []map[string]any
}

func (bi *batchInput) getForMarshalling() any {
	return bi.inputs
}

type testCase struct {
	name    string
	inputs  []TestInput
	outputs []map[string]any

	inputNodes  []yaml.Node
	outputNodes []yaml.Node
}

func (tc *testCase) GetName() string {
	return tc.name
}

func (tc *testCase) GetInputs() []TestInput {
	return tc.inputs
}

func (tc *testCase) GetOutputs() []map[string]any {
	return tc.outputs
}

type test struct {
	fileName string
	name     string

	testCases  []testCase
	inputChan  chan TestInput
	outputChan chan string

	executableDone chan struct{}

	beforeEach []func() error
	afterEach  []func() error
	logging    bool
}

func (t *test) GetTestCases() []TestCase {
	testCases := make([]TestCase, len(t.testCases))
	for i, tc := range t.testCases {
		testCases[i] = &tc
	}

	return testCases
}

func (t *test) SetLogging(logging bool) {
	t.logging = logging
}

func (tst *test) Run(t TestRunner) {
	for _, f := range tst.beforeEach {
		err := f()
		if err != nil {
			t.Errorf("error in before each: %v", err)
		}
	}

	t.Run(tst.name, func(t TestRunner) {
		for _, tc := range tst.testCases {
			t.Run(tc.name, func(t TestRunner) {
				for _, in := range tc.inputs {
					// send the input to the input channel
					tst.inputChan <- in
				}

				for i, out := range tc.outputs {
					expectedJsonB, err := json.Marshal(out)
					if err != nil {
						t.Errorf("error marshalling expected output: %v", err)
					}
					expectedJson := string(expectedJsonB)
					if tst.logging {
						t.Logf("expecting output: %s", expectedJson)
					}

					// receive the output from the output channel
					// and block until we get the output
					// or until underlying process finishes
					select {
					case <-tst.executableDone:
						// the process has finished, but output is not yet available
						t.Errorf("process finished before output %d was received, expected: %s", i, expectedJson)
						return
					case got := <-tst.outputChan:
						var out map[string]any
						err = json.Unmarshal([]byte(got), &out)
						if err != nil {
							t.Errorf("error unmarshalling output: %v", err)
							if tst.logging {
								t.Logf("failed on output: %s", got)
							}
						}

						expectedNode := tc.outputNodes[i]
						if assertMatch(t, &expectedNode, out) {
							if tst.logging {
								t.Logf("output matches: %s", got)
							}
						}
					}
				}
			})
		}
	})

	for _, f := range tst.afterEach {
		err := f()
		if err != nil {
			t.Errorf("error in after each: %v", err)
		}
	}
}

func (tst *test) read(path string) error {
	testFile, err := os.Open(path)
	if err != nil {
		return err
	}

	return tst.readFromReader(testFile)
}

func (tst *test) readFromReader(thereader io.Reader) error {
	// Test is a multi-document yaml file.
	// Example file content:
	//
	// case: list_current_dir_files
	// # input 1
	// in_: {"jsonrpc": "2.0", "method": "Files.List", "params": {"path": "."}, "id": 1}
	// # output 1
	// out: {"jsonrpc": "2.0", "result": ["in.yaml"], "id": 1}
	// ---
	//
	// # input 2
	// in_1: {"jsonrpc": "2.0", "method": "Files.List", "params": {"path": "/"}, "id": 2}
	// in_2: {"jsonrpc": "2.0", "method": "Files.List", "params": {"path": "/"}, "id": 2}
	// # output 2
	// out: {"jsonrpc": "2.0", "result": ["bin", "dev", "etc", "home"], "id": 2}
	// ---
	// # and so on
	//
	//

	decoder := yaml.NewDecoder(thereader)
	for {
		var docNode yaml.Node
		err := decoder.Decode(&docNode)

		// var doc map[string]interface{}
		// err := decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		var doc map[string]interface{}
		err = docNode.Decode(&doc)
		if err != nil {
			return err
		}

		var intermediateDoc map[string]yaml.Node
		err = docNode.Decode(&intermediateDoc)
		if err != nil {
			return err
		}

		var tc testCase
		if doc["case"] != nil {
			tc.name = doc["case"].(string)
		}

		for key, value := range doc {
			if strings.HasPrefix(key, "in") {
				testInputBatch, ok := value.([]interface{})
				if ok {
					testInputBatchMap := make([]map[string]interface{}, len(testInputBatch))
					for i, v := range testInputBatch {
						testInput, ok := v.(map[string]interface{})
						if !ok {
							return fmt.Errorf("%w for test '%s', expected map for key '%s', but got '%T': %v", ErrFailedToParseInput, tst.name, key, v, v)
						}
						testInputBatchMap[i] = testInput
					}
					tc.inputs = append(tc.inputs, &batchInput{
						inputs: testInputBatchMap,
					})
				} else {
					testInput, ok := value.(map[string]interface{})
					if !ok {
						return fmt.Errorf("%w for test '%s' , expected map or array of maps for key '%s', but got '%T': %v", ErrFailedToParseInput, tst.name, key, value, value)
					}
					tc.inputs = append(tc.inputs, &singleInput{
						input: testInput,
					})
				}
				if inNode, ok := intermediateDoc[key]; ok {
					tc.inputNodes = append(tc.inputNodes, inNode)
				}

			} else if strings.HasPrefix(key, "out") {
				testOutput, ok := value.(map[string]interface{})
				if !ok {
					return fmt.Errorf("%w for test '%s' , expected map for key '%s', but got '%T': %v", ErrFailedToParseOutput, tst.name, key, value, value)
				}
				tc.outputs = append(tc.outputs, testOutput)
				if outNode, ok := intermediateDoc[key]; ok {
					tc.outputNodes = append(tc.outputNodes, outNode)
				}
			}
		}

		tst.testCases = append(tst.testCases, tc)
	}

	return nil
}

// Read would read a collection of tests from a folder
// specified by path
func Read(path string) (TestSuite, error) {
	suite := &testSuite{
		inputChan:      make(chan TestInput),
		outputChan:     make(chan string),
		testsDone:      make(chan struct{}),
		completed:      make(chan struct{}),
		executableDone: make(chan struct{}),
		path:           path,
	}

	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() { // skip directories, we only take the first level files
			continue
		}

		suffix := "_test.yaml"

		if strings.HasSuffix(file.Name(), suffix) {
			name := strings.TrimSuffix(file.Name(), suffix)
			test := &test{
				fileName: file.Name(),
				name:     name,

				inputChan:  suite.inputChan,
				outputChan: suite.outputChan,

				beforeEach: suite.beforeEach,
				afterEach:  suite.afterEach,

				executableDone: suite.executableDone,

				logging: suite.logging,
			}
			path := filepath.Join(dir.Name(), file.Name())
			err := test.read(path)
			if err != nil {
				return nil, err
			}
			suite.tests = append(suite.tests, test)
		}
	}

	return suite, nil
}

func (ts *testSuite) pipeInput(t TestRunner) {
	ts.transport.pipeInput(t, ts)
}

func (ts *testSuite) pipeOutput(t TestRunner) {
	ts.transport.pipeOutput(t, ts)
}

func (ts *testSuite) startExecutable(t TestRunner) *exec.Cmd {
	//nolint:gosec // #nosec G204 -- user gives us this command to run, so they must be sure to want to run it
	cmd := exec.Command(ts.command, ts.args...)
	addToGroup(cmd)

	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("error creating stdin pipe: %v", err)
	}
	ts.targetInput = in

	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("error creating stdout pipe: %v", err)
	}
	ts.targetOutput = out

	// cmd.Stderr = os.Stderr
	// Tried to pipe stderr, so everything logged to stderr simply goes to the test output
	// This, however, makes certain CI runs in Github Actions fail, with this error:
	// exec: WaitDelay expired before I/O complete
	// and this only happens when using go test ./... , while not happening with simple go test
	// So would instead send stderr to the test log output

	if ts.logging {
		stderrPipeCreated := make(chan struct{})
		go func() {
			var errReader io.ReadCloser
			var scanner *bufio.Scanner
			createPipe := func() {
				defer close(stderrPipeCreated)
				var err error
				errReader, err = cmd.StderrPipe()
				if err != nil {
					t.Fatalf("error creating stderr pipe: %v", err)
				}
				scanner = bufio.NewScanner(errReader)
			}
			createPipe()

			// log while tests are running

			for {
				select {
				case <-ts.executableDone:
					return
				default:
					if !scanner.Scan() {
						return
					}
					t.Logf("stderr: %s", scanner.Text())
				}
			}
		}()
		<-stderrPipeCreated
	}

	if ts.logging {
		t.Logf("running command: %s %s", ts.command, strings.Join(ts.args, " "))
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("error running command: %v", err)
	}

	go func() {
		defer close(ts.executableDone)
		if ts.logging {
			t.Log("waiting for command to finish")
		}

		if err := cmd.Wait(); err != nil {
			t.Logf("error waiting for command to finish: %v", err)
		}
		if ts.logging {
			t.Log("command finished")
		}
	}()

	return cmd
}

func (ts *testSuite) setup(t TestRunner) {
	processDone := make(chan struct{})
	if ts.command == "" {
		t.Fatalf("no executable specified, make sure to call WithExecutable before running the test suite")
		return
	} else {
		cmd := ts.startExecutable(t)
		go func() {
			defer close(processDone)
			// kill the process when the test suite is done
			<-ts.testsDone

			if ts.logging {
				t.Log("stop executable")
			}
			if err := stop(cmd.Process); err != nil {
				t.Logf("error stopping process with SIGTERM, trying to kill it: %v", err)
				// fallback to killing the process
				if err := cmd.Process.Kill(); err != nil {
					t.Logf("error killing process: %v", err)
				}
			}
			if ts.logging {
				t.Log("executable stopped")
			}
		}()
	}
	inPipeFinished := make(chan struct{})
	go func() {
		defer close(inPipeFinished)
		ts.pipeInput(t)
	}()

	outPipeFinished := make(chan struct{})
	go func() {
		defer close(outPipeFinished)
		ts.pipeOutput(t)
	}()

	go func() {
		defer close(ts.completed)

		<-ts.testsDone
		// wait for all pipes to finish
		<-inPipeFinished
		<-outPipeFinished

		// wait for the process to finish
		<-processDone
		<-ts.executableDone
	}()
}
