package server

import (
	"context"

	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/session"
)

type Transport interface {
	Run(
		capabilities *mcp.ServerCapabilities,
		serverInfo *mcp.Implementation,
		serverOptions ...ServerOption,
	) error

	Shutdown(context.Context) error

	GetSessionManager() *session.SessionManager
}
