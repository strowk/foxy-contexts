package stdio

import (
	"io"

	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
)

type StdioTransportOption interface {
	apply(*stdioTransport)
}

type StdioTransportNewServerOption struct {
	NewServer func(
		capabilities *mcp.ServerCapabilities,
		serverInfo *mcp.Implementation,
		options ...server.ServerOption,
	) server.Server
}

func (o *StdioTransportNewServerOption) apply(s *stdioTransport) {
	s.newServer = o.NewServer
}

func WithNewServerFunc(newServer func(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
	options ...server.ServerOption,
) server.Server) StdioTransportOption {
	return &StdioTransportNewServerOption{
		NewServer: newServer,
	}
}

type stdioOutOption struct {
	out io.Writer
}

func (o *stdioOutOption) apply(s *stdioTransport) {
	s.out = o.out
}

func WithOut(out io.Writer) StdioTransportOption {
	return &stdioOutOption{
		out: out,
	}
}

type stdioInOption struct {
	in io.Reader
}

func (o *stdioInOption) apply(s *stdioTransport) {
	s.in = o.in
}

func WithIn(in io.Reader) StdioTransportOption {
	return &stdioInOption{
		in: in,
	}
}
