package stdio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"github.com/strowk/foxy-contexts/pkg/session"
)

func NewTransport(options ...StdioTransportOption) server.Transport {
	tp := &stdioTransport{
		shuttingDown:            make(chan struct{}),
		stoppedReadingResponses: make(chan struct{}),
		stoppedReadingInput:     make(chan struct{}),
		stopped:                 make(chan struct{}),

		in:  os.Stdin,
		out: os.Stdout,

		newServer: func(
			capabilities *mcp.ServerCapabilities,
			serverInfo *mcp.Implementation,
			options ...server.ServerOption,
		) server.Server {
			return server.NewServer(capabilities, serverInfo, options...)
		},

		sessionManager: session.NewSessionManager(),
	}

	for _, o := range options {
		o.apply(tp)
	}

	return tp
}

type stdioTransport struct {
	shuttingDown            chan struct{}
	stoppedReadingResponses chan struct{}
	stoppedReadingInput     chan struct{}
	stopped                 chan struct{}

	in  io.Reader
	out io.Writer

	newServer func(
		capabilities *mcp.ServerCapabilities,
		serverInfo *mcp.Implementation,
		options ...server.ServerOption,
	) server.Server

	sessionManager *session.SessionManager
}

func (s *stdioTransport) Run(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
	options ...server.ServerOption,
) error {
	server := s.newServer(capabilities, serverInfo, options...)
	return s.run(server)
}

func (s *stdioTransport) run(
	srv server.Server,
) error {
	// local stdio transport is using only one session per whole execution
	ctx := context.Background()
	ctx, _, err := s.sessionManager.CreateNewSession(ctx, nil)
	if err != nil {
		srv.GetLogger().LogEvent(foxyevent.FailedCreatingSession{Err: err})
		return fmt.Errorf("failed to create session: %w", err)
	}

	reader := bufio.NewReader(s.in)
	go func() {
		defer close(s.stoppedReadingResponses)
	out:
		for {
			select {
			case <-s.shuttingDown:
				break out
			case res := <-srv.GetResponses():
				data, err := jsonrpc2.Marshal(res.Id, res.Result, res.Error)
				if err != nil {
					srv.GetLogger().LogEvent(foxyevent.StdioFailedMarhalResponse{Err: err})
				}
				srv.GetLogger().LogEvent(foxyevent.StdioSendingResponse{Data: data})
				_, err = s.out.Write(data)
				if err != nil {
					srv.GetLogger().LogEvent(foxyevent.StdioFailedWriting{Err: err})
					break out
				}
				_, err = s.out.Write([]byte("\n"))
				if err != nil {
					srv.GetLogger().LogEvent(foxyevent.StdioFailedWriting{Err: err})
					break out
				}
			}
		}
	}()

	go func() {
		defer close(s.stoppedReadingInput)
	out:
		for {
			input, err := reader.ReadBytes('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					srv.GetLogger().LogEvent(foxyevent.StdioFailedReadingInput{Err: err})
				}
				break out
			}
			srv.Handle(ctx, input)
		}
	}()

	// wait for either shutting down or stopped reading input
	select {
	case <-s.shuttingDown:
	case <-s.stoppedReadingInput:
		// if we stopped reading input, we can now initiate transport shutdown
		close(s.shuttingDown)
	}

	// wait until we stop reading responses,
	// before signaling that we are stopped
	<-s.stoppedReadingResponses

	close(s.stopped)
	return nil
}

func (s *stdioTransport) Shutdown(ctx context.Context) error {
	safeClose(s.shuttingDown)

	// this waits either till we are stopped or we cannot wait anymore
	select {
	case <-s.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func safeClose(ch chan struct{}) {
	defer func() {
		//nolint:errcheck // it is ok to ignore if there was no panic
		recover()
		// it is possible to panic if channel is already closed
		// in which case we just go on, as it is possible that
		// Shutdown would be called soon after transport is stopped
	}()
	close(ch)
}

func (s *stdioTransport) GetSessionManager() *session.SessionManager {
	return s.sessionManager
}
