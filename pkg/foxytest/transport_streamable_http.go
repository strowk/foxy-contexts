package foxytest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/strowk/foxy-contexts/pkg/sse"
)

type testTransportStreamableHTTP struct {
	url string

	httpClient *http.Client

	postResponses chan *response
}

type response struct {
	body        []byte
	contentType string
}

func NewTestTransportStreamableHTTP(url string) *testTransportStreamableHTTP {
	client := &http.Client{}

	return &testTransportStreamableHTTP{
		url:        url,
		httpClient: client,

		postResponses: make(chan *response),
	}
}

func (t *testTransportStreamableHTTP) ping(tr TestRunner) error {
	req, err := http.NewRequest("POST", t.url, bytes.NewReader([]byte(`{"method":"ping","params":{},"id":0, "jsonrpc":"2.0"}`)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			tr.Logf("error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error pinging streamable http: %s", resp.Status)
	}

	return nil
}

func (t *testTransportStreamableHTTP) pipeInput(tr TestRunner, ts *testSuite) {
	var retries = 0
	retriesLimit := 3
	for {
		respErr := t.ping(tr)

		if respErr != nil {
			if ts.logging {
				tr.Logf("error pinging: %v, retried %d/%d", respErr, retries, retriesLimit)
			}
			// retrying because servers often are not ready when their process is only started
			// and streamable http protocol does not provide any official way to probe the server
			if retries < retriesLimit {
				retries++
				time.Sleep(time.Duration(math.Round(300+100*math.Pow(float64(retries), 2))) * time.Millisecond)
				continue
			} else {
				tr.Errorf("failed to ping starting server: %v, retries exhausted", respErr)
				return
			}
		}
		if ts.logging {
			tr.Logf("server %s pinged successfully", t.url)
		}
		break
	}

	for {
		select {
		case <-ts.testsDone:
			close(t.postResponses)
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

			bodyReader := bytes.NewReader(data)

			req, err := http.NewRequest("POST", t.url, bodyReader)
			if err != nil {
				tr.Errorf("error creating request: %v", err)
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")

			// TODO: figure out how to define session id
			// req.Header.Set("Mcp-Session-Id", ts.sessionId.String())

			resp, err := t.httpClient.Do(req)
			if err != nil {
				tr.Errorf("error sending request: %v", err)
			}
			defer func() {
				err := resp.Body.Close()
				if err != nil {
					tr.Logf("error closing response body: %v", err)
				}
			}()

			// read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				tr.Errorf("error reading response body: %v", err)
			}

			respForChannel := &response{
				body:        body,
				contentType: resp.Header.Get("Content-Type"),
			}

			t.postResponses <- respForChannel

			if ts.logging {
				var cutBody = false
				var bodyBytesToLog = body

				if len(body) > 1024 {
					bodyBytesToLog = body[:1024]
					cutBody = true
				}

				bodyToLog := string(bodyBytesToLog)
				if cutBody {
					bodyToLog += "... truncated"
				}

				tr.Logf("received response: %s, content-type: %s body: \n%s", resp.Status, respForChannel.contentType, bodyToLog)
			}

			if resp.StatusCode != http.StatusOK {
				tr.Errorf("error sending request: %v", resp.Status)
			}
		}
	}
}

func (t *testTransportStreamableHTTP) pipeOutput(tr TestRunner, ts *testSuite) {
	if ts.logging {
		tr.Log("reading post responses")
	}
	for resp := range t.postResponses {

		byteBody := resp.body

		if resp.contentType == "application/json" {
			// send the output to the output channel
			ts.outputChan <- string(byteBody)
		} else if resp.contentType == "text/event-stream" {
			// parse the event stream and send every event to the output channel
			reader := bufio.NewReader(bytes.NewReader(byteBody))
		decodingLoop:
			for {
				event, err := sse.DecodeEvent(reader)
				if err != nil {
					if err == io.EOF {
						break decodingLoop
					}
					tr.Errorf("error decoding event: %v", err)
					break decodingLoop
				}
				if len(event.Data) > 0 {

					// TODO: figure out how to timeout here, this can be stuck in case if noone is reading from channel
					// to reproduce this, remove "out_2:" from /examples/streamable_http/testdata/list_tools_and_promts_test.yaml
					ts.outputChan <- string(event.Data)
				} else {
					tr.Errorf("error decoding event: no data found")
				}
			}
		} else {
			tr.Errorf("unknown content type: %s", resp.contentType)
		}
	}

	if ts.logging {
		tr.Log("finished reading post responses")
	}
}
