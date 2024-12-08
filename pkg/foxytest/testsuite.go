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

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestRunner is an interface that is compatible with the testing.T interface
// , so that the test suite can be run with the standard go test runner
// or with any other runner that implements this interface, particularly
// it is used by foxytest own command line runner allowing to use this
// test suite runner from languages other than Go
type TestRunner interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Run(name string, f func(t TestRunner)) bool
}

type stdRunner struct {
	t *testing.T
}

func NewTestRunner(t *testing.T) TestRunner {
	return &stdRunner{
		t: t,
	}
}

func (tc *stdRunner) Fatal(args ...interface{}) {
	tc.t.Fatal(args...)
}

func (tc *stdRunner) Fatalf(format string, args ...interface{}) {
	tc.t.Fatalf(format, args...)
}

func (tc *stdRunner) Errorf(format string, args ...interface{}) {
	tc.t.Errorf(format, args...)
}

func (tc *stdRunner) Log(args ...interface{}) {
	tc.t.Log(args...)
}

func (tc *stdRunner) Logf(format string, args ...interface{}) {
	tc.t.Logf(format, args...)
}

func (tc *stdRunner) Run(name string, f func(t TestRunner)) bool {
	f(tc)
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

	inputChan  chan map[string]interface{}
	outputChan chan string

	// targetInput is to write to the target process, where the SuT takes input
	targetInput io.Writer

	// targetOutput is to read from the target process, where the SuT writes output
	targetOutput io.Reader

	command string
	args    []string

	errors []error

	logging bool

	testsDone chan struct{}
	completed chan struct{}
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

func (ts *testSuite) AssertNoErrors(t TestRunner) {
	if len(ts.errors) > 0 {
		t.Errorf("errors: %v", ts.errors)
	}
}

func (ts *testSuite) Errors() []error {
	return ts.errors
}

func (ts *testSuite) Run(t TestRunner) {
	// firstly run all before all functions,
	// returning if any of them fails
	for _, f := range ts.beforeAll {
		err := f()
		if err != nil {
			ts.errors = append(ts.errors, err)
			return
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

	// signal that we are done
	close(ts.testsDone)

	if len(testErrors) > 0 {
		ts.errors = append(ts.errors, testErrors...)
	}

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
	GetInputs() []map[string]interface{}
	GetOutputs() []map[string]interface{}
}

type testCase struct {
	name    string
	inputs  []map[string]interface{}
	outputs []map[string]interface{}
}

func (tc *testCase) GetName() string {
	return tc.name
}

func (tc *testCase) GetInputs() []map[string]interface{} {
	return tc.inputs
}

func (tc *testCase) GetOutputs() []map[string]interface{} {
	return tc.outputs
}

type test struct {
	fileName string
	name     string

	testCases  []testCase
	inputChan  chan map[string]interface{}
	outputChan chan string

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

				for _, out := range tc.outputs {
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
					got := <-tst.outputChan

					// TODO: do we really need to unmarshal the output? it is not like it is going to be used
					var out map[string]interface{}
					err = json.Unmarshal([]byte(got), &out)
					if err != nil {
						t.Errorf("error unmarshalling output: %v", err)
					}

					if assert.JSONEq(t, expectedJson, got) {
						if tst.logging {
							t.Logf("output matches: %s", got)
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
	// out: {"jsonrpc": "2.0", "result": ["bin", "dev", "etc", "home", "lib", "media", "mnt", "opt", "proc", "root", "run", "sbin", "srv", "sys", "tmp", "usr", "var"], "id": 2}
	// ---
	// # and so on
	//
	//

	testFile, err := os.Open(path)
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(testFile)
	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		var tc testCase
		if doc["case"] != nil {
			tc.name = doc["case"].(string)
		}

		for key, value := range doc {
			if strings.HasPrefix(key, "in") {
				testInput, ok := value.(map[string]interface{})
				if !ok {
					return fmt.Errorf("failed to parse input for test '%s' , expected map for key '%s', but got '%T': %v", tst.name, key, value, value)
				}
				tc.inputs = append(tc.inputs, testInput)
			} else if strings.HasPrefix(key, "out") {
				testOutput, ok := value.(map[string]interface{})
				if !ok {
					return fmt.Errorf("failed to parse output for test '%s' , expected map for key '%s', but got '%T': %v", tst.name, key, value, value)
				}
				tc.outputs = append(tc.outputs, testOutput)
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
		inputChan:  make(chan map[string]interface{}),
		outputChan: make(chan string),
		testsDone:  make(chan struct{}),
		completed:  make(chan struct{}),
		path:       path,
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
	for {
		select {
		case <-ts.testsDone:
			if ts.logging {
				t.Log("finished reading inputs")
			}
			return
		case in := <-ts.inputChan:
			// write the input to the target process
			// where the SuT takes input
			data, err := json.Marshal(in)
			if err != nil {
				t.Errorf("error marshalling input: %v", err)
			}

			if ts.logging {
				t.Logf("sending input: %s", string(data))
			}
			_, err = ts.targetInput.Write(data)
			if err != nil {
				t.Errorf("error writing input: %v", err)
			}
			_, err = ts.targetInput.Write([]byte("\n"))
			if err != nil {
				t.Errorf("error writing input: %v", err)
			}
		}
	}
}

func (ts *testSuite) pipeOutput(t TestRunner) {
	// read the output from the target process
	// where the SuT writes output
	reader := bufio.NewReader(ts.targetOutput)
	for {

		finish := false
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			t.Logf("error reading output: %v", err)
			return
		}
		if err == io.EOF {
			if ts.logging {
				t.Log("finished reading output")
			}
			finish = true
		}

		if line != "" {
			// send the output to the output channel
			ts.outputChan <- line
		}

		if finish {
			break
		}
	}
}

func (ts *testSuite) startExecutable(t TestRunner) *exec.Cmd {
	cmd := exec.Command(ts.command, ts.args...)
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

	go func() {
		errReader, err := cmd.StderrPipe()
		if err != nil {
			t.Fatalf("error creating stderr pipe: %v", err)
		}
		scanner := bufio.NewScanner(errReader)
		for scanner.Scan() {
			t.Logf("stderr: %s", scanner.Text())
		}
		if ts.logging {
			t.Log("finished reading stderr")
		}
	}()

	if ts.logging {
		t.Logf("running command: %s %s", ts.command, strings.Join(ts.args, " "))
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("error running command: %v", err)
	}

	go func() {
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
			t.Log("executable stopped")
			close(processDone)
		}()
	}
	inPipeFinished := make(chan struct{})
	go func() {
		ts.pipeInput(t)
		close(inPipeFinished)
	}()

	outPipeFinished := make(chan struct{})
	go func() {
		ts.pipeOutput(t)
		close(outPipeFinished)
	}()

	go func() {
		<-ts.testsDone
		// wait for all pipes to finish
		<-inPipeFinished
		<-outPipeFinished

		// wait for the process to finish
		<-processDone

		close(ts.completed)
	}()
}
