package server

import (
	"log/slog"

	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

type Server interface {
	Handle(b []byte)
	GetResponses() chan jsonrpc2.JsonRpcResponse
	SetRequestHandler(request jsonrpc2.Request, handler func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error))
	SetNotificationHandler(request jsonrpc2.Request, handler func(req jsonrpc2.Request))
	SetLogger(logger foxyevent.Logger)
	GetLogger() foxyevent.Logger
}

type server struct {
	router    jsonrpc2.JsonRpcRouter
	responses chan jsonrpc2.JsonRpcResponse
	logger    foxyevent.Logger
}

func NewServer(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
	options ...ServerOption,
) Server {
	s := &server{
		router:    jsonrpc2.NewJsonRPCRouter(),
		responses: make(chan jsonrpc2.JsonRpcResponse),
		logger:    foxyevent.NewSlogLogger(slog.Default()),
	}

	appliedNotificationHandler := false
	for _, o := range options {
		if _, ok := o.(InitializationFininshedHandlerOption); ok {
			appliedNotificationHandler = true
		}
		o.apply(s)
	}

	if !appliedNotificationHandler {
		s.SetNotificationHandler(&mcp.InitializedNotification{}, func(req jsonrpc2.Request) {
			// just ignore it to not return any errors
		})
	}

	s.initialize(capabilities, serverInfo)

	return s
}

func (s *server) SetRequestHandler(request jsonrpc2.Request, handler func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error)) {
	s.router.SetRequestHandler(request, handler)
}

func (s *server) SetNotificationHandler(request jsonrpc2.Request, handler func(req jsonrpc2.Request)) {
	s.router.SetNotificationHandler(request, handler)
}

func (s *server) initialize(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
) {
	s.SetRequestHandler(&mcp.InitializeRequest{},
		func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
			return s.handleInitialize(req, capabilities, serverInfo), nil
		},
	)
	s.SetRequestHandler(&mcp.PingRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		return struct{}{}, nil
	})
}

func (*server) handleInitialize(
	req jsonrpc2.Request,
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
) jsonrpc2.Result {
	requestedVersion := req.(*mcp.InitializeRequest).Params.ProtocolVersion

	// will pick either the requested version or the highest supported version
	supportedVersion := requestedVersion
	for _, version := range SUPPORTED_PROTOCOL_VERSIONS {
		supportedVersion = version
		if version == requestedVersion {
			break
		}
	}

	return &mcp.InitializeResult{
		ProtocolVersion: supportedVersion,
		Capabilities:    *capabilities,
		ServerInfo:      *serverInfo,
	}
}

func (s *server) GetResponses() chan jsonrpc2.JsonRpcResponse {
	return s.responses
}

func (s *server) Handle(buffer []byte) {
	res := s.router.Handle(buffer)
	if res != nil {
		s.responses <- *res
	}
}

func (s *server) SetLogger(logger foxyevent.Logger) {
	s.logger = logger
}

func (s *server) GetLogger() foxyevent.Logger {
	return s.logger
}
