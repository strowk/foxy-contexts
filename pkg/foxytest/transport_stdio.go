package foxytest

import (
	"bufio"
	"encoding/json"
	"io"
)

type TestTransportStdio struct{}

func (t *TestTransportStdio) pipeInput(tr TestRunner, ts *testSuite) {
	for {
		select {
		case <-ts.testsDone:
			if ts.logging {
				tr.Log("finished reading inputs")
			}
			return
		case in := <-ts.inputChan:
			// write the input to the target process
			// where the SuT takes input
			data, err := json.Marshal(in.getForMarshalling())
			if err != nil {
				tr.Errorf("error marshalling input: %v", err)
			}

			if ts.logging {
				tr.Logf("sending input: %s", string(data))
			}
			_, err = ts.targetInput.Write(data)
			if err != nil {
				tr.Errorf("error writing input: %v", err)
			}
			_, err = ts.targetInput.Write([]byte("\n"))
			if err != nil {
				tr.Errorf("error writing input: %v", err)
			}
		}
	}
}

func (t *TestTransportStdio) pipeOutput(tr TestRunner, ts *testSuite) {
	// read the output from the target process
	// where the SuT writes output
	reader := bufio.NewReader(ts.targetOutput)
	for {

		finish := false
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			tr.Logf("error reading output: %v", err)
			return
		}
		if err == io.EOF {
			if ts.logging {
				tr.Log("finished reading output")
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
