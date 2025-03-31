package foxytest

type TestTransportType string

const (
	TestTransportTypeStdio          TestTransportType = "stdio"
	TestTransportTypeStreamableHTTP TestTransportType = "streamable_http"
)

type TestTransport interface {
	pipeInput(tr TestRunner, ts *testSuite)
	pipeOutput(tr TestRunner, ts *testSuite)
}
