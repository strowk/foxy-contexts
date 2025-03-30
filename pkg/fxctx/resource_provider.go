package fxctx

import (
	"context"

	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type ResourceProvider interface {
	GetResources(ctx context.Context) ([]mcp.Resource, error)
	ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
}

type resourceProvider struct {
	getResources func(ctx context.Context) ([]mcp.Resource, error)
	readResource func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
}

func (r *resourceProvider) GetResources(ctx context.Context) ([]mcp.Resource, error) {
	return r.getResources(ctx)
}

func (r *resourceProvider) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	return r.readResource(ctx, uri)
}

func NewResourceProvider(
	getResources func(ctx context.Context) ([]mcp.Resource, error),
	readResource func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error),
) ResourceProvider {
	return &resourceProvider{
		getResources: getResources,
		readResource: readResource,
	}
}

func AsResourceProvider(f any) any {
	return fx.Annotate(f, fx.As(new(ResourceProvider)), fx.ResultTags(`group:"resource_providers"`))
}
