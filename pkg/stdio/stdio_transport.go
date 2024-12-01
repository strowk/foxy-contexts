package stdio

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
)

func NewTransport() server.Transport {
	tp := &stdioTransport{
		shuttingDown:            make(chan struct{}),
		stoppedReadingResponses: make(chan struct{}),
		stoppedReadingInput:     make(chan struct{}),
		stopped:                 make(chan struct{}),
	}

	return tp
}

type stdioTransport struct {
	shuttingDown            chan struct{}
	stoppedReadingResponses chan struct{}
	stoppedReadingInput     chan struct{}
	stopped                 chan struct{}
}

func (s *stdioTransport) Run(
	capabilities mcp.ServerCapabilities,
	serverInfo mcp.Implementation,
	options ...server.ServerOption,
) error {
	server := server.NewServer(capabilities, serverInfo, options...)
	return s.run(server)
}

func (s *stdioTransport) run(
	srv server.Server,
) error {
	reader := bufio.NewReader(os.Stdin)

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
				os.Stdout.Write(data)
				os.Stdout.Write([]byte("\n"))
			}
		}
		close(s.stoppedReadingResponses)
	}()

	go func() {
	out:
		for {
			select {
			case <-s.shuttingDown:
				close(s.stoppedReadingInput)
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
