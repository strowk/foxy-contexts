package streamable_http

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
)

type streamableHttpTransport struct {
	e *echo.Echo
}

func (s *streamableHttpTransport) Run(capabilities *mcp.ServerCapabilities, serverInfo *mcp.Implementation, serverOptions ...server.ServerOption) error {
	panic("unimplemented")
}

func (s *streamableHttpTransport) Shutdown(context.Context) error {
	panic("unimplemented")
}

func NewTransport() server.Transport {
	return &streamableHttpTransport{}
}
