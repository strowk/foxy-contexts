package stdio

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"

	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
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
	reader := bufio.NewReader(s.in)
	go func() {
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
		close(s.stoppedReadingResponses)
	}()

	go func() {
	out:
		for {
			select {
			case <-s.shuttingDown:
				break out
			default:
				input, err := reader.ReadBytes('\n')
				if err != nil {
					if !errors.Is(err, io.EOF) {
						srv.GetLogger().LogEvent(foxyevent.StdioFailedReadingInput{Err: err})
					}
					break out
				}
				srv.Handle(input)
			}
		}
		close(s.stoppedReadingInput)
	}()

	<-s.shuttingDown
	<-s.stoppedReadingResponses
	<-s.stoppedReadingInput

	close(s.stopped)

	return nil
}

func (s *stdioTransport) Shutdown(ctx context.Context) error {
	close(s.shuttingDown)

	select {
	case <-s.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
