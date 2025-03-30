package fxctx

import (
	"context"

	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type Resource interface {
	GetResource(ctx context.Context) mcp.Resource
	ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
}

type resource struct {
	mcp.Resource
	readFunc func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
}

func (t *resource) GetResource(ctx context.Context) mcp.Resource {
	return t.Resource
}

func (t *resource) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	return t.readFunc(ctx, uri)
}

func NewResource(
	mcpResource mcp.Resource,
	callback func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)) Resource {
	return &resource{
		Resource: mcpResource,
		readFunc: callback,
	}
}

func AsResource(f any) any {
	return fx.Annotate(f, fx.As(new(Resource)), fx.ResultTags(`group:"resources"`))
}
