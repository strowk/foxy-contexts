package streamable_http

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/sse"
)

func TestStreamableHttpTransport(t *testing.T) {
	tr := NewTransport(
		Endpoint{
			Hostname: "localhost",
			Port:     8080,
			Path:     "/mcp",
		})

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go func() {
		assert.EqualError(t, tr.Run(&mcp.ServerCapabilities{}, &mcp.Implementation{
			Name:    "TestServer",
			Version: "0.0.0",
		}), "http: Server closed")
		waitGroup.Done()
	}()

	defer func() {
		assert.NoError(t, tr.Shutdown(context.Background()))
		waitGroup.Wait()
	}()

	// testing https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/transports/#listening-for-messages-from-the-server
	// for now we only check that the server is responding with 405 as spec says it should when "long-running" SSE is not supported
	t.Run("GET is not yet supported", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:8080/mcp", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	// testing https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/transports/#sending-messages-to-the-server
	// with one resoponse

	t.Run("POST ping once", func(t *testing.T) {
		body := `{"method":"ping","params":{},"id":0, "jsonrpc":"2.0"}`
		req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		require.JSONEq(t, `{"jsonrpc":"2.0","result":{},"id":0}`, string(respBody))
	})

	// testing https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/transports/#sending-messages-to-the-server
	// with several responses

	t.Run("POST ping batch", func(t *testing.T) {
		body := `[{"method":"ping","params":{},"id":1, "jsonrpc":"2.0"},{"method":"ping","params":{},"id":2, "jsonrpc":"2.0"}]`
		req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		require.NoError(t, err)
		bodyReader := bufio.NewReader(bytes.NewBuffer(respBody))

		event1, err := sse.DecodeEvent(bodyReader)
		if !assert.NoError(t, err) {
			t.Fatalf("failed to decode event: %s", string(respBody))
		}
		require.Equal(t, `{"jsonrpc":"2.0","result":{},"id":1}`, string(event1.Data))

		event2, err := sse.DecodeEvent(bodyReader)
		if !assert.NoError(t, err) {
			t.Fatalf("failed to decode event: %s", string(respBody))
		}
		require.Equal(t, `{"jsonrpc":"2.0","result":{},"id":2}`, string(event2.Data))
	})

	t.Run("POST ping twice with same session", func(t *testing.T) {
		body := `{"method":"ping","params":{},"id":0, "jsonrpc":"2.0"}`
		req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		require.JSONEq(t, `{"jsonrpc":"2.0","result":{},"id":0}`, string(respBody))
		sessionId := resp.Header.Get("MCP-Session-Id")
		require.NotEmpty(t, sessionId)

		body = `{"method":"ping","params":{},"id":1, "jsonrpc":"2.0"}`
		req, err = http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Session-Id", sessionId)
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		require.JSONEq(t, `{"jsonrpc":"2.0","result":{},"id":1}`, string(respBody))
	})

	t.Run("POST lifecycle", func(t *testing.T) {
		body := `{
			"id":0, 
			"jsonrpc":"2.0",
			"method":"initialize",
			"params":{
				"protocolVersion": "2025-03-26",
				"capabilities": {},
				"clientInfo": {
					"name": "TestClient",
					"version": "0.0.0"
				}
			}
		}`
		req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		require.JSONEq(t, `{
			"id":0,
			"jsonrpc":"2.0",
			"result": {
				"capabilities": {}, 
				"protocolVersion":"2025-03-26", 
				"serverInfo": {"name":"TestServer", "version":"0.0.0"}
			}
		}`, string(respBody))
		sessionId := resp.Header.Get("MCP-Session-Id")
		require.NotEmpty(t, sessionId)

		body = `{"method":"notifications/initialized","params":{},"jsonrpc":"2.0"}`
		req, err = http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(body)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Session-Id", sessionId)
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		respBody, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		require.Equal(t, "", resp.Header.Get("Content-Type"))
		require.Equal(t, ``, string(respBody))
	})

}

func TestMarshalServerError(t *testing.T) {
	r := &jsonrpc2.JsonRpcResponse{
		Id: jsonrpc2.NewIntRequestId(1),
	}

	testCases := []struct {
		name     string
		id       jsonrpc2.RequestId
		expected string
	}{
		{
			name:     "null id",
			id:       jsonrpc2.NewNullRequestId(),
			expected: `{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error","data":"assert.AnError general error for testing"},"id":null}`,
		},
		{
			name:     "int id",
			id:       jsonrpc2.NewIntRequestId(1),
			expected: `{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error","data":"assert.AnError general error for testing"},"id":1}`,
		},
		{
			name:     "string id",
			id:       jsonrpc2.NewStringRequestId("1"),
			expected: `{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error","data":"assert.AnError general error for testing"},"id":"1"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r.Id = tc.id
			m := marshalServerError(r, assert.AnError)
			assert.JSONEq(t, tc.expected, string(m))
		})
	}
}
