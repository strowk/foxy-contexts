package streamable_http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grokify/go-pkce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strowk/foxy-contexts/pkg/auth"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/sse"
)

func TestStreamableHttpTransportWithAuth(t *testing.T) {
	tr := NewTransport(
		Endpoint{
			Hostname: "localhost",
			Port:     8080,
			Path:     "/mcp",
		},
	)

	oauth2Auth, err := auth.NewOauth2Authorization(
		auth.WithAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
			return "test", nil
		}),
	)
	require.NoError(t, err)
	tr.(auth.AuthorizeableTransport).PlugInAuthorization(oauth2Auth)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go func() {
		assert.EqualError(t, tr.Run(&mcp.ServerCapabilities{}, &mcp.Implementation{
			Name:    "TestServer",
			Version: "0.0.0",
		}), "http: Server closed")
		waitGroup.Done()
	}()

	t.Cleanup(func() {
		assert.NoError(t, tr.Shutdown(context.Background()))
		waitGroup.Wait()
	})

	// addresses race condition between server start and test start
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := http.Get("http://localhost:8080/mcp")
		assert.NoError(c, err)
		defer func() { assert.NoError(c, resp.Body.Close()) }()
	}, 5*time.Second, 200*time.Millisecond)

	pingBody := `{"method":"ping","params":{},"id":0, "jsonrpc":"2.0"}`
	t.Run("POST /mcp without authorization", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewReader([]byte(pingBody)))
		assert.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		bodyStr := string(body)
		require.Equal(t, "authorization header is missing", bodyStr)
	})

	t.Run("GET oauth protected resource metadata", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:8080/.well-known/oauth-protected-resource", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		body, err := io.ReadAll(resp.Body)
		bodyStr := string(body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t, `{
			"resource":"https://localhost:8080/mcp",
			"authorization_servers": ["https://localhost:8080"]
		}`, bodyStr)
	})

	t.Run("GET oauth discovery", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:8080/.well-known/oauth-authorization-server", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		body, err := io.ReadAll(resp.Body)
		bodyStr := string(body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.JSONEq(t, `{
			"issuer":"https://localhost:8080/mcp",
			"authorization_endpoint":"https://localhost:8080/authorize",
			"token_endpoint":"https://localhost:8080/token",
			"registration_endpoint":"https://localhost:8080/register",
			"scopes_supported":[],
			"response_types_supported":["code"],
			"response_modes_supported":["form_post"],
			"grant_types_supported":["authorization_code"],
			"token_endpoint_auth_methods_supported":["client_secret_post"],
			"token_endpoint_auth_signing_alg_values_supported":["RS256"],
			"code_challenge_methods_supported":["S256"]
		}`, bodyStr)
	})

	var clientId string
	var clientSecret string

	t.Run("register oauth 2 client", func(t *testing.T) {
		registerClientBody := `{
			"client_name": "test-client-name",
			"redirect_uris": ["http://localhost:8081/callback"],
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"]
		}`

		reqBody := []byte(registerClientBody)

		req, err := http.NewRequest("POST", "http://localhost:8080/register", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		body, err := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var registeredClient struct {
			ClientId     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}

		err = json.Unmarshal(body, &registeredClient)
		require.NoError(t, err)
		clientId = registeredClient.ClientId
		clientSecret = registeredClient.ClientSecret

		t.Log(clientSecret)
	})

	var codeVerifier string
	var codeForToken string
	t.Run("GET oauth /authorize", func(t *testing.T) {
		codeVerifier, err = pkce.NewCodeVerifier(-1)
		require.NoError(t, err)
		codeChallenge := pkce.CodeChallengeS256(codeVerifier)
		authorizeUrl := fmt.Sprintf(`http://localhost:8080/authorize?client_id=%s&response_type=code&code_challenge=%s&redirect_uri=http://localhost:8081/callback&code_challenge_method=S256`, clientId, codeChallenge)
		req, err := http.NewRequest("GET", authorizeUrl, nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		body, err := io.ReadAll(resp.Body)
		bodyStr := string(body)
		assert.Equal(t, http.StatusFound, resp.StatusCode)
		assert.Empty(t, bodyStr)
		location := resp.Header.Get("location")
		assert.Contains(t, location, "http://localhost:8081/callback?code=")
		codeForToken = strings.Replace(location, "http://localhost:8081/callback?code=", "", 1)
	})

	var accessToken string
	t.Run("POST for oauth2 token", func(t *testing.T) {
		resp, err := http.PostForm("http://localhost:8080/token", url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {codeForToken},
			"redirect_uri":  {"http://localhost:8081/callback"},
			"code_verifier": {codeVerifier},
			"client_id":     {clientId},
			"client_secret": {clientSecret},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		parsedToken := new(struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int    `json:"expires_in"`
			RefreshToken string `json:"refresh_token"`
		})

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		err = json.Unmarshal(body, parsedToken)
		require.NoError(t, err)

		require.NotEmpty(t, parsedToken.AccessToken)
		require.Equal(t, "Bearer", parsedToken.TokenType)
		require.Equal(t, parsedToken.ExpiresIn, 7200)
		require.NotEmpty(t, parsedToken.RefreshToken)
		accessToken = parsedToken.AccessToken
	})

	t.Run("POST /mcp with authorization", func(t *testing.T) {
		req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewReader([]byte(pingBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() { assert.NoError(t, resp.Body.Close()) }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		bodyStr := string(body)
		require.JSONEq(t, `{
			"jsonrpc":"2.0",
			"result":{},
			"id":0
		}`, bodyStr)
	})
}

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

	// addresses race condition between server start and test start
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := http.Get("http://localhost:8080/mcp")
		assert.NoError(c, err)
		defer func() { assert.NoError(c, resp.Body.Close()) }()
	}, 5*time.Second, 200*time.Millisecond)

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
