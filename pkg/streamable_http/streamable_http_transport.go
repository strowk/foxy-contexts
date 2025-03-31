package streamable_http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/session"
	"github.com/strowk/foxy-contexts/pkg/sse"
)

type streamableHttpTransport struct {
	e *echo.Echo

	keepStreamAliveInterval time.Duration

	port     int
	hostname string
	path     string

	sessionManager *session.SessionManager
}

func (t *streamableHttpTransport) Run(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
	serverOptions ...server.ServerOption,
) error {

	e := echo.New()
	t.e = e

	servers := sync.Map{}

	// ensure that negotiated version would be at least the one with streamable http transport
	serverOptions = append(serverOptions, server.MinimalProtocolVersionOption{
		Version: server.MINIMAL_FOR_STREAMABLE_HTTP,
	})

	e.DELETE(t.path, func(c echo.Context) error {
		sessionIdHeader := c.Request().Header.Get("Mcp-Session-Id")
		if sessionIdHeader == "" {
			return echo.NewHTTPError(400, "Mcp-Session-Id header is required")
		}
		sessionId, err := uuid.Parse(sessionIdHeader)
		if err != nil {
			return echo.NewHTTPError(400, "Wrong session id format, expected UUID")
		}
		_, ok := servers.Load(sessionId)
		if !ok {
			return echo.NewHTTPError(404, "Requested session id not found in session store")
		}
		servers.Delete(sessionId)
		t.sessionManager.DeleteSession(sessionId)
		return c.NoContent(204)
	})

	e.POST(t.path, func(c echo.Context) error {
		w := c.Response()
		var serv server.Server
		var sessionIdUsed uuid.UUID
		if c.Request().Header.Get("Mcp-Session-Id") != "" {
			sessionId, err := uuid.Parse(c.Request().Header.Get("Mcp-Session-Id"))
			if err != nil {
				// wrong session id format is equivalent to not finding the session
				// , hence we return 404 Not Found with some details in the body
				return echo.NewHTTPError(404, "Wrong session id format, expected UUID")
			}
			s, ok := servers.Load(sessionId)
			if !ok {
				return echo.NewHTTPError(404, "Requested session id not found in session store")
			}
			w.Header().Set("Mcp-Session-Id", sessionId.String())
			serv = s.(server.Server)
			sessionIdUsed = sessionId
		} else {
			sessionId := uuid.New()
			s := server.NewServer(capabilities, serverInfo, serverOptions...)
			servers.Store(sessionId, s)
			w.Header().Set("Mcp-Session-Id", sessionId.String())
			serv = s
			sessionIdUsed = sessionId
		}

		w.Header().Set("MCP-Session-Id", sessionIdUsed.String())
		ctx, _, err := t.sessionManager.ResolveSessionOrCreateNew(c.Request().Context(), sessionIdUsed)
		if err != nil {
			return echo.NewHTTPError(404, "Failed to resolve session")
		}

		buf, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(500, "Failed to read request body")
		}

		responses := serv.HandleAndGetResponses(ctx, buf)
		if len(responses) == 0 {
			return c.NoContent(202)
		}

		if len(responses) == 1 {
			if responses[0] == nil {
				w.WriteHeader(202)
				return nil
			}
			return c.JSON(200, responses[0])
		}

		// Multiple resonses have to be marshalled as event stream

		hasAnyResponses := false
		for _, r := range responses {
			if r == nil {
				continue // notificiation processed
			}
			m, err := json.Marshal(r)
			if err != nil {
				m = marshalServerError(r, err)
			}
			ev := sse.Event{
				Data: m,
			}
			if !hasAnyResponses {
				// must write the header before the first event
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(200)
			}
			err = ev.MarshalTo(w)
			if err != nil {
				serv.GetLogger().LogEvent(foxyevent.StreamingHTTPFailedMarshalEvent{
					Err: err,
				})
			}
			hasAnyResponses = true
		}
		if !hasAnyResponses {
			w.WriteHeader(202)
		}
		return nil
	})

	return e.Start(fmt.Sprintf("%s:%d", t.hostname, t.port))
}

func marshalServerError(r *jsonrpc2.JsonRpcResponse, e error) []byte {
	id := r.Id

	if r.Id.IdIsMissing {
		id = jsonrpc2.NewNullRequestId()
	}

	srvErr := jsonrpc2.NewServerError(-32000, e.Error())
	srvErrStr, e := jsonrpc2.Marshal(id, nil, srvErr)
	if e != nil {
		panic(e)
	}

	return srvErrStr
}

func (t *streamableHttpTransport) Shutdown(ctx context.Context) error {
	return t.e.Shutdown(ctx)
}

func NewTransport(options ...TransportOption) server.Transport {
	tp := &streamableHttpTransport{
		keepStreamAliveInterval: 5 * time.Second,

		path:     "/mcp",
		hostname: "127.0.0.1",
		port:     8080,

		sessionManager: session.NewSessionManager(),
	}
	for _, o := range options {
		o.apply(tp)
	}
	return tp
}

type TransportOption interface {
	apply(*streamableHttpTransport)
}

func (s *streamableHttpTransport) GetSessionManager() *session.SessionManager {
	return s.sessionManager
}
