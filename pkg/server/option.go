package server

import (
	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

type ServerOption interface {
	apply(Server)
}

type ServerStartCallbackOption struct {
	Callback func(s Server)
}

func (o ServerStartCallbackOption) apply(s Server) {
	o.Callback(s)
}

type LoggerOption struct {
	Logger foxyevent.Logger
}

func (o LoggerOption) apply(s Server) {
	s.SetLogger(o.Logger)
}

type InitializationFininshedHandlerOption struct {
	callback func(req *mcp.InitializedNotification)
}

func (o InitializationFininshedHandlerOption) apply(s Server) {
	s.SetNotificationHandler(&mcp.InitializedNotification{}, func(req jsonrpc2.Request) {
		o.callback(req.(*mcp.InitializedNotification))
	})
}
