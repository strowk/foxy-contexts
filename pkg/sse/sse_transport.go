package sse

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	// "github.com/labstack/echo/v4/middleware"
)

type SSETransportOption interface {
	apply(*sseTransport)
}

func NewTransport(options ...SSETransportOption) server.Transport {
	tp := &sseTransport{
		keepAliveInterval: 5 * time.Second,
	}

	for _, o := range options {
		o.apply(tp)
	}

	return tp
}

type sseTransport struct {
	keepAliveInterval time.Duration
	e                 *echo.Echo
}

func newResponseEvent(res jsonrpc2.JsonRpcResponse) (*Event, error) {
	data, err := jsonrpc2.Marshal(res.Id, res.Result, res.Error)
	if err != nil {
		return nil, err
	}

	return &Event{
		Event: []byte("message"),
		Data:  data,
	}, nil
}

func (s *sseTransport) Run(
	capabilities mcp.ServerCapabilities,
	serverInfo mcp.Implementation,
	options ...server.ServerOption,
) error {
	e := echo.New()
	s.e = e

	// e.Use(middleware.Logger())

	servers := map[uuid.UUID]server.Server{}

	postEndpoint := "/message"

	e.GET("/sse", func(c echo.Context) error {
		sessionId := uuid.New()
		srv := server.NewServer(capabilities, serverInfo, options...)
		servers[sessionId] = srv
		w := c.Response()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		event := Event{
			Event: []byte("endpoint"),
			Data:  []byte(postEndpoint + "?sessionId=" + sessionId.String()),
		}
		if err := event.MarshalTo(w); err != nil {
			return err
		}
		w.Flush()

		ticker := time.NewTicker(s.keepAliveInterval)
		defer ticker.Stop()
		srv.GetLogger().LogEvent(foxyevent.SSEClientConnected{ClientIP: c.RealIP()})

		shuttingDown := make(chan struct{})
		e.Server.RegisterOnShutdown(func() {
			close(shuttingDown)
		})

		for {
			select {
			case <-shuttingDown:
				// Protocol does not seem to have a way to notify client about server initiated shutdown
				// so we would just close the connection to allow server to shutdown and client to reconnect
				// to, hopefully, a new server instance started by orchestrator
				delete(servers, sessionId)
				return nil
			case <-c.Request().Context().Done():
				delete(servers, sessionId)
				srv.GetLogger().LogEvent(foxyevent.SSEClientDisconnected{ClientIP: c.RealIP()})
				return nil
			case res := <-srv.GetResponses():
				event, err := newResponseEvent(res)
				if err != nil {
					srv.GetLogger().LogEvent(foxyevent.SSEFailedCreatingEvent{Err: err})
					continue
				}
				if err := event.MarshalTo(w); err != nil {
					srv.GetLogger().LogEvent(foxyevent.SSEFailedMarshalEvent{Err: err})
				}
				w.Flush()
			case <-ticker.C:
				event := CommentEvent{
					Comment: []byte("keep-alive"),
				}
				if err := event.MarshalTo(w); err != nil {
					return err
				}
				w.Flush()
			}
		}
	})

	e.POST(postEndpoint, func(c echo.Context) error {
		sessionId := c.QueryParams().Get("sessionId")
		if sessionId == "" {
			return c.String(http.StatusBadRequest, "sessionId is required")
		}
		parsedSessionId, err := uuid.Parse(sessionId)
		if err != nil {
			return c.String(http.StatusBadRequest, "sessionId is not a valid UUID")
		}

		r, ok := servers[parsedSessionId]
		if !ok {
			return c.String(http.StatusNotFound, "session not found")
		}

		b, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, "failed to read request body")
		}

		r.Handle(b)
		return c.JSON(http.StatusAccepted, "Accepted")
	})

	return e.Start("127.0.0.1:1323")
}

func (s *sseTransport) Shutdown(ctx context.Context) error {
	if s.e != nil {
		return s.e.Shutdown(ctx)
	}
	return nil
}
