package server

import (
	"context"
	"log/slog"

	foxyevent "github.com/strowk/foxy-contexts/pkg/foxy_event"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

type Server interface {
	Handle(ctx context.Context, b []byte)
	HandleAndGetResponses(ctx context.Context, b []byte) []*jsonrpc2.JsonRpcResponse
	GetResponses() chan jsonrpc2.JsonRpcResponse
	SetRequestHandler(request jsonrpc2.Request, handler func(ctx context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error))
	SetNotificationHandler(request jsonrpc2.Request, handler func(ctx context.Context, req jsonrpc2.Request))
	SetLogger(logger foxyevent.Logger)
	GetLogger() foxyevent.Logger
}

type server struct {
	router    jsonrpc2.JsonRpcRouter
	responses chan jsonrpc2.JsonRpcResponse
	logger    foxyevent.Logger

	minimalProtocolVersionOption *MinimalProtocolVersionOption
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
		s.SetNotificationHandler(&mcp.InitializedNotification{}, func(_ context.Context, _ jsonrpc2.Request) {
			// just ignore it to not return any errors
		})
	}

	s.initialize(capabilities, serverInfo)

	return s
}

func (s *server) SetRequestHandler(request jsonrpc2.Request, handler func(ctx context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error)) {
	s.router.SetRequestHandler(request, func(ctx context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		return handler(ctx, req)
	})
}

func (s *server) SetNotificationHandler(request jsonrpc2.Request, handler func(ctx context.Context, req jsonrpc2.Request)) {
	s.router.SetNotificationHandler(request,
		func(ctx context.Context, req jsonrpc2.Request) {
			handler(ctx, req)
		})
}

func (s *server) initialize(
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
) {
	if capabilities == nil {
		panic("capabilities cannot be nil")
	}
	if serverInfo == nil {
		panic("serverInfo cannot be nil")
	}
	s.SetRequestHandler(&mcp.InitializeRequest{},
		func(_ context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
			return s.handleInitialize(req, capabilities, serverInfo), nil
		},
	)
	s.SetRequestHandler(&mcp.PingRequest{}, func(_ context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		return struct{}{}, nil
	})
}

func (s *server) handleInitialize(
	req jsonrpc2.Request,
	capabilities *mcp.ServerCapabilities,
	serverInfo *mcp.Implementation,
) jsonrpc2.Result {
	requestedVersion := req.(*mcp.InitializeRequest).Params.ProtocolVersion

	// minimal version is either set or defaults to empty with every supprorted version
	// being higher than empty string
	var minimal string
	if s.minimalProtocolVersionOption != nil {
		minimal = string(s.minimalProtocolVersionOption.Version)
	}

	// will pick either the requested version or the highest supported version
	supportedVersion := requestedVersion
	for _, version := range SUPPORTED_PROTOCOL_VERSIONS {
		versionStr := string(version)
		supportedVersion = versionStr
		if (versionStr == requestedVersion) && (versionStr >= minimal) {
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

func (s *server) Handle(ctx context.Context, buffer []byte) {
	responses := s.router.Handle(ctx, buffer)
	for _, response := range responses {
		if response != nil {
			s.responses <- *response
		}
	}
}

func (s *server) HandleAndGetResponses(ctx context.Context, buffer []byte) []*jsonrpc2.JsonRpcResponse {
	return s.router.Handle(ctx, buffer)
}

func (s *server) SetLogger(logger foxyevent.Logger) {
	s.logger = logger
}

func (s *server) GetLogger() foxyevent.Logger {
	return s.logger
}
