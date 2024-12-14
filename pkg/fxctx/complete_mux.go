package fxctx

import (
	"errors"
	"fmt"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

type Completer interface {
	Complete(*mcp.CompleteRequest, string) (*mcp.CompleteResult, error)
}

type CompleteMux interface {
	Complete(*mcp.CompleteRequest) (*mcp.CompleteResult, error)
	RegisterHandlers(s server.Server)
}

var (
	ErrMissingRefType        = errors.New("missing ref type")
	ErrUnknownValueOfRefType = errors.New("unknown value of ref type")
	ErrUnknownTypeOfRef      = errors.New("unknown type of ref")
)

type completeMux struct {
	promptMux   Completer
	resourceMux Completer
}

func (c *completeMux) Complete(req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	if ref, ok := req.Params.Ref.(map[string]interface{}); ok {
		return c.complete(req, ref)
	} else {
		return nil, fmt.Errorf("%w: %T", ErrUnknownTypeOfRef, req.Params.Ref)
	}
}

func (c *completeMux) complete(req *mcp.CompleteRequest, ref map[string]interface{}) (*mcp.CompleteResult, error) {
	if refType, ok := ref["type"]; ok {
		switch refType {
		case "ref/prompt":
			return c.completePrompt(req, ref)
		case "ref/resource":
			return c.completeResource(req, ref)
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnknownValueOfRefType, refType)
		}
	} else {
		return nil, fmt.Errorf("%w: no type in %v", ErrMissingRefType, ref)
	}
}

func (c *completeMux) completePrompt(req *mcp.CompleteRequest, ref map[string]interface{}) (*mcp.CompleteResult, error) {
	if promptName, ok := ref["name"].(string); ok {
		return c.promptMux.Complete(req, promptName)
	} else {
		return nil, fmt.Errorf("%w: %T", ErrUnknownTypeOfRef, ref["name"])
	}
}

func (c *completeMux) completeResource(req *mcp.CompleteRequest, ref map[string]interface{}) (*mcp.CompleteResult, error) {
	if uri, ok := ref["uri"].(string); ok {
		return c.resourceMux.Complete(req, uri)
	} else {
		return nil, fmt.Errorf("%w: %T", ErrUnknownTypeOfRef, ref["uri"])
	}
}

type CompletMuxParams struct {
	fx.In

	ResourceMux ResourceMux `optional:"true"`
	PromptMux   PromptMux   `optional:"true"`
}

func NewCompleteMux(params CompletMuxParams) CompleteMux {
	return &completeMux{
		promptMux:   params.PromptMux,
		resourceMux: params.ResourceMux,
	}
}

func (c *completeMux) RegisterHandlers(s server.Server) {
	c.setCompletionCompleteHandler(s)
}

func (c *completeMux) setCompletionCompleteHandler(s server.Server) {
	s.SetRequestHandler(&mcp.CompleteRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		r := req.(*mcp.CompleteRequest)
		res, err := c.Complete(r)
		if err != nil {
			return nil, jsonrpc2.NewServerError(CompleteFailed, fmt.Sprintf("failed to complete: %v", err.Error()))
		}
		return res, nil
	})
}

func ProvideCompleteMux() fx.Option {
	return fx.Provide(NewCompleteMux)
}
