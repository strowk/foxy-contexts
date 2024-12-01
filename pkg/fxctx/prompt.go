package fxctx

import (
	"fmt"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

type Prompt interface {
	GetMcpPrompt() mcp.Prompt
	Get(*mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
}

type prompt struct {
	mcp.Prompt
	get func(*mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
}

func (p *prompt) GetMcpPrompt() mcp.Prompt {
	return p.Prompt
}

func (p *prompt) Get(req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return p.get(req)
}

func NewPrompt(
	p mcp.Prompt,
	get func(*mcp.GetPromptRequest) (*mcp.GetPromptResult, error),
) Prompt {
	return &prompt{
		Prompt: p,
		get:    get,
	}
}

func AsPrompt(f any) any {
	return fx.Annotate(f, fx.As(new(Prompt)), fx.ResultTags(`group:"prompts"`))
}

type PromptMux interface {
	ListPrompts() []mcp.Prompt
	RegisterHandlers(s server.Server)
}

type promptMux struct {
	prompts map[string]Prompt
}

func NewPromptMux(prompts []Prompt) PromptMux {
	promptsMap := make(map[string]Prompt)
	for _, p := range prompts {
		promptsMap[p.GetMcpPrompt().Name] = p
	}
	return &promptMux{
		prompts: promptsMap,
	}
}

func (p *promptMux) ListPrompts() []mcp.Prompt {
	var prompts []mcp.Prompt
	for _, p := range p.prompts {
		prompts = append(prompts, p.GetMcpPrompt())
	}
	return prompts
}

func (p *promptMux) GetPrompt(
	req *mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	prompt, ok := p.prompts[req.Params.Name]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", req.Params.Name)
	}

	return prompt.Get(req)
}

func ProvidePromptMux() fx.Option {
	return fx.Provide(fx.Annotate(
		NewPromptMux,
		fx.ParamTags(`group:"prompts"`),
	))
}

func (p *promptMux) RegisterHandlers(s server.Server) {
	p.setListPromptsHandler(s)
	p.setGetPromptHandler(s)
}

func (p *promptMux) setListPromptsHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ListPromptsRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		resp := &mcp.ListPromptsResult{
			Prompts: []mcp.Prompt{},
		}

		list := p.ListPrompts()
		for _, p := range list {
			resp.Prompts = append(resp.Prompts, p)
		}

		return resp, nil
	})
}

func (p *promptMux) setGetPromptHandler(s server.Server) {
	s.SetRequestHandler(&mcp.GetPromptRequest{}, func(req jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		r := req.(*mcp.GetPromptRequest)
		res, err := p.GetPrompt(r)
		if err != nil {
			return nil, jsonrpc2.NewServerError(GetPromptFailed, fmt.Sprintf("failed to get prompt: %v", err.Error()))
		}
		return res, nil
	})
}
