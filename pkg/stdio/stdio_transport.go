package stdio

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
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
		log.Printf("stopped reading responses")
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

	// wait for either shutting down or stopped reading input
	select {
	case <-s.shuttingDown:
		// if we got shutting down signal,
		// we need to wait until we stop reading input,
		// which waits for shutdown by itself
		<-s.stoppedReadingInput
	case <-s.stoppedReadingInput:
		// if we stopped reading input, we can now initiate shutdown
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
		if r := recover(); r != nil {
			// it is possible to panic if channel is already closed
			// in which case we just go on, as it is possible that
			// Shutdown would be called soon after transport is stopped
		}
	}()
	close(ch)
}
