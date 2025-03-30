package server

import (
	"context"

	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

type ServerOption interface {
	apply(*server)
}

type ServerStartCallbackOption struct {
	Callback func(s Server)
}

func (o ServerStartCallbackOption) apply(s *server) {
	o.Callback(s)
}

type LoggerOption struct {
	Logger foxyevent.Logger
}

func (o LoggerOption) apply(s *server) {
	s.SetLogger(o.Logger)
}

type InitializationFininshedHandlerOption struct {
	callback func(req *mcp.InitializedNotification)
}

func (o InitializationFininshedHandlerOption) apply(s *server) {
	s.SetNotificationHandler(&mcp.InitializedNotification{}, func(ctx context.Context, req jsonrpc2.Request) {
		o.callback(req.(*mcp.InitializedNotification))
	})
}

type MinimalProtocolVersionOption struct {
	Version ProtocolVersion
}

func (minimalVersionOption MinimalProtocolVersionOption) apply(s *server) {
	s.minimalProtocolVersionOption = &minimalVersionOption
}
