package fxctx

import (
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type Resource interface {
	GetResource() mcp.Resource
	ReadResource(uri string) mcp.ReadResourceResult
}

type resource struct {
	mcp.Resource
	readFunc func(uri string) mcp.ReadResourceResult
}

func (t *resource) GetResource() mcp.Resource {
	return t.Resource
}

func (t *resource) ReadResource(uri string) mcp.ReadResourceResult {
	return t.readFunc(uri)
}

func NewResource(
	mcpResource mcp.Resource,
	callback func(uri string) mcp.ReadResourceResult) Resource {
	return &resource{
		Resource: mcpResource,
		readFunc: callback,
	}
}

func AsResource(f any) any {
	return fx.Annotate(f, fx.As(new(Resource)), fx.ResultTags(`group:"resources"`))
}
