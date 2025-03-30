package fxctx

import (
	"context"
	"fmt"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

type ResourceMux interface {
	Completer
	GetResources(ctx context.Context) ([]mcp.Resource, error)
	ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
	RegisterHandlers(s server.Server)
}

type resourceMux struct {
	resources         map[string]Resource
	resourceProviders []ResourceProvider
}

func (m *resourceMux) Complete(ctx context.Context, req *mcp.CompleteRequest, uri string) (*mcp.CompleteResult, error) {
	// TODO: this has to be implemented when the URI templates are implemented
	return &mcp.CompleteResult{
		Completion: mcp.CompleteResultCompletion{
			HasMore: utils.Ptr(false),
			Total:   utils.Ptr(0),
			Values:  []string{},
		},
	}, nil
}

func NewResourceMux(
	resources []Resource,
	resourceProviders []ResourceProvider,
) ResourceMux {
	m := map[string]Resource{}

	for _, res := range resources {
		m[res.GetResource(context.Background()).Uri] = res
	}

	return &resourceMux{
		resources:         m,
		resourceProviders: resourceProviders,
	}
}

func (m *resourceMux) GetResources(ctx context.Context) ([]mcp.Resource, error) {
	res := []mcp.Resource{}

	for _, r := range m.resources {
		res = append(res, r.GetResource(ctx))
	}

	for _, provider := range m.resourceProviders {
		provided, err := provider.GetResources(ctx)
		if err != nil {
			return nil, err
		}
		res = append(res, provided...)
	}

	return res, nil
}

func (m *resourceMux) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	res, ok := m.resources[uri]
	if !ok {
		for _, p := range m.resourceProviders {
			foundResource, err := p.ReadResource(ctx, uri)
			if err != nil {
				return nil, err
			}
			if foundResource != nil {
				return foundResource, nil
			}
		}

		return &mcp.ReadResourceResult{}, nil
	}

	return res.ReadResource(ctx, uri)
}

func (m *resourceMux) RegisterHandlers(s server.Server) {
	m.setResourceListHandler(s)
	m.setReadResourceHandler(s)
}

func (m *resourceMux) setResourceListHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ListResourcesRequest{}, func(ctx context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		resp := &mcp.ListResourcesResult{
			Resources: []mcp.Resource{},
		}

		list, err := m.GetResources(ctx)
		if err != nil {
			return nil, jsonrpc2.NewServerError(ListResourcesFailed, fmt.Sprintf("failed to get resources: %v", err.Error()))
		}

		resp.Resources = append(resp.Resources, list...)

		return resp, nil
	})
}

func (m *resourceMux) setReadResourceHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ReadResourceRequest{}, func(ctx context.Context, req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		r := req.(*mcp.ReadResourceRequest)
		res, err := m.ReadResource(ctx, r.Params.Uri)
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
