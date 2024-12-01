package fxctx

import (
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type ResourceProvider interface {
	GetResources() ([]mcp.Resource, error)
	ReadResource(uri string) (*mcp.ReadResourceResult, error)
}

type resourceProvider struct {
	getResources func() ([]mcp.Resource, error)
	readResource func(uri string) (*mcp.ReadResourceResult, error)
}

func (r *resourceProvider) GetResources() ([]mcp.Resource, error) {
	return r.getResources()
}

func (r *resourceProvider) ReadResource(uri string) (*mcp.ReadResourceResult, error) {
	return r.readResource(uri)
}

func NewResourceProvider(
	getResources func() ([]mcp.Resource, error),
	readResource func(uri string) (*mcp.ReadResourceResult, error),
) ResourceProvider {
	return &resourceProvider{
		getResources: getResources,
		readResource: readResource,
	}
}

func AsResourceProvider(f any) any {
	return fx.Annotate(f, fx.As(new(ResourceProvider)), fx.ResultTags(`group:"resource_providers"`))
}
