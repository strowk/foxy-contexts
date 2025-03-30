package foxytest

import (
	"bytes"
	"sync"
	"testing"

	testrunner_mock "github.com/strowk/foxy-contexts/pkg/foxytest/mock"
	"go.uber.org/mock/gomock"
)

// internalTestRunner is a wrapper around mock test runner
// and the testing.T, which delegates all calls to the mock
// test runner, except for Run that works the same as standard
// runner by delegating to the testing.T.Run, which will cause
// test to run in a separate goroutine, while logs, failures
// and errors will be captured by the mock test runner, which
// will allow us to verify how the test package worked
type internalTestRunner struct {
	themock *testrunner_mock.MockTestSystem

	t *testing.T
}

//go:generate mockgen -destination=./mock/testrunner_mock.go -package=testrunner_mock . TestSystem

func (tc *internalTestRunner) Fatal(args ...any) {
	tc.themock.Fatal(args...)
}

func (tc *internalTestRunner) Fatalf(format string, args ...any) {
	tc.themock.Fatalf(format, args...)
}

func (tc *internalTestRunner) Errorf(format string, args ...any) {
	tc.t.Helper()
	tc.themock.Errorf(format, args...)
}

func (tc *internalTestRunner) Log(args ...any) {
	tc.themock.Log(args...)
}

func (tc *internalTestRunner) Logf(format string, args ...any) {
	tc.themock.Logf(format, args...)
}

func (tc *internalTestRunner) Run(name string, f func(t TestRunner)) bool {
	tc.t.Run(name, func(t *testing.T) {
		f(tc)
	})
	return true
}

func createTestTest() *test {
	// input channel we would need to have test to write stuff in
	inputChan := make(chan TestInput)

	// the output channel we would be writing to to see what test would do
	outputChan := make(chan string)

	// this indicates for the test that the executable has finished and there is no more output to expect
	executableDone := make(chan struct{})

	// create a new test
	thetest := &test{
		name:           "test",
		inputChan:      inputChan,
		outputChan:     outputChan,
		executableDone: executableDone,
	}
	return thetest
}

// TestTheTest is testing.. a test so that we ensure
// that the testing package does things we expect it to do
func TestTheTest(t *testing.T) {
	mockController := gomock.NewController(t)
	testSystemMock := testrunner_mock.NewMockTestSystem(mockController)
	runner := &internalTestRunner{
		themock: testSystemMock,
		t:       t,
	}

	testCases := []struct {
		name         string
		caseDocument string
		output       string
		expectError  bool
	}{
		{
			name: "with regex",
			caseDocument: `
case: case 1
in: {}
out: {"val": 1, "reg": !!re "[\\w]{1,5}!"}
`,
			output: `{"val": 1, "reg": "haha!"}`,
		},
		{
			name: "with regex failing",
			caseDocument: `
case: case 1
in: {}
out: {"val": 1, "reg": !!re "^[\\w]{1,5}!$"}
`,
			output:      `{"val": 1, "reg": "hohoho!, oh no"}`,
			expectError: true,
		},
		{
			name: "with embedded regex",
			caseDocument: `
case: case 1
in: {}
out: {"val": 1, "reg": !!ere "this is usual string, /and this is regex [\\w]{1,3}/ ! .. end"}
`,
			output: `{"val": 1, "reg": "this is usual string, and this is regex yay ! .. end"}`,
		},
		{
			name: "with embedded regex, escaping slashes",
			caseDocument: `
case: case 1
in: {}
out: {"val": 1, "reg": !!ere "this is usual string escaping \\/some slashes, /and this is regex [\\w]{1,3}/ ! .. end"}
`, // note that in sequence \\/ second backslash is needed ^ here to escape second backslash through YAML,
			// then that second backslash is needed to escape forward slash from embedded regex
			output: `{"val": 1, "reg": "this is usual string escaping /some slashes, and this is regex WOW ! .. end"}`,
		},
		{
			name: "with batch input",
			caseDocument: `
case: case 1
in: [{"val": "hoho"}, {"val": "hohoho"}]
out: {"val": "whatever"}
`,
			output: `{"val": "whatever"}`,
		},
	}

	for _, tc := range testCases {
		buf := []byte(tc.caseDocument)

		thetest := createTestTest()
		defer close(thetest.inputChan)
		defer close(thetest.outputChan)
		defer close(thetest.executableDone)

		err := thetest.readFromReader(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("failed to read test from reader: %v", err)
		}

		waitForInput := sync.WaitGroup{}
		waitForInput.Add(1)
		go func() {
			defer waitForInput.Done()
			<-thetest.inputChan
		}()

		if tc.expectError {
			testSystemMock.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
		}

		waitForTest := sync.WaitGroup{}
		waitForTest.Add(1)
		go func() {
			defer waitForTest.Done()
			// run the test
			thetest.Run(runner)
		}()

		waitForInput.Wait() // we need to wait for input to unblock the test to get to the output
		thetest.outputChan <- tc.output
		waitForTest.Wait() // this should not hang, since all outputs are written

	}
}
