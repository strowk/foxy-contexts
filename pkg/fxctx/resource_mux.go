package fxctx

import (
	"fmt"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

type ResourceMux interface {
	GetResources() ([]mcp.Resource, error)
	ReadResource(uri string) (*mcp.ReadResourceResult, error)
	RegisterHandlers(s server.Server)
}

type resourceMux struct {
	resources         map[string]Resource
	resourceProviders []ResourceProvider
}

func NewResourceMux(
	resources []Resource,
	resourceProviders []ResourceProvider,
) ResourceMux {
	m := map[string]Resource{}

	for _, res := range resources {
		m[res.GetResource().Uri] = res
	}

	return &resourceMux{
		resources:         m,
		resourceProviders: resourceProviders,
	}
}

func (m *resourceMux) GetResources() ([]mcp.Resource, error) {
	res := []mcp.Resource{}

	for _, r := range m.resources {
		res = append(res, r.GetResource())
	}

	for _, provider := range m.resourceProviders {
		provided, err := provider.GetResources()
		if err != nil {
			return nil, err
		}
		for _, r := range provided {
			res = append(res, r)
		}
	}

	return res, nil
}

func (m *resourceMux) ReadResource(uri string) (*mcp.ReadResourceResult, error) {
	res, ok := m.resources[uri]
	if !ok {
		for _, p := range m.resourceProviders {
			foundResource, err := p.ReadResource(uri)
			if err != nil {
				return nil, err
			}
			if foundResource != nil {
				return foundResource, nil
			}
		}

		return &mcp.ReadResourceResult{}, nil
	}

	return res.ReadResource(uri)
}

func (m *resourceMux) RegisterHandlers(s server.Server) {
	m.setResourceListHandler(s)
	m.setReadResourceHandler(s)
}

func (m *resourceMux) setResourceListHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ListResourcesRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		resp := &mcp.ListResourcesResult{
			Resources: []mcp.Resource{},
		}

		list, err := m.GetResources()
		if err != nil {
			return nil, jsonrpc2.NewServerError(ListResourcesFailed, fmt.Sprintf("failed to get resources: %v", err.Error()))
		}

		for _, res := range list {
			resp.Resources = append(resp.Resources, res)
		}

		return resp, nil
	})
}

func (m *resourceMux) setReadResourceHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ReadResourceRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		r := req.(*mcp.ReadResourceRequest)
		res, err := m.ReadResource(r.Params.Uri)
		if err != nil {
			return nil, jsonrpc2.NewServerError(ReadResourceFailed, fmt.Sprintf("failed to read resource: %v", err.Error()))
		}
		return &res, nil
	})
}

func ProvideResourceMux() fx.Option {
	return fx.Provide(fx.Annotate(
		NewResourceMux,
		fx.ParamTags(`group:"resources"`, `group:"resource_providers"`),
	))
}
