package fxctx

import (
	"errors"
	"fmt"

	"github.com/strowk/foxy-contexts/internal/utils"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type CompleterFunc = func(arg *mcp.PromptArgument, value string) (*mcp.CompleteResult, error)

type Prompt interface {
	GetMcpPrompt() mcp.Prompt
	Get(*mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
	Complete(*mcp.CompleteRequest) (*mcp.CompleteResult, error)
	WithCompleter(CompleterFunc) Prompt
}

type prompt struct {
	mcp.Prompt
	get       func(*mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
	completer CompleterFunc
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

func (p *prompt) WithCompleter(c CompleterFunc) Prompt {
	p.completer = c
	return p
}

func AsPrompt(f any) any {
	return fx.Annotate(f, fx.As(new(Prompt)), fx.ResultTags(`group:"prompts"`))
}

var (
	ErrNoSuchArgument = errors.New("no such argument to complete for prompt")
)

func (p *prompt) Complete(req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	for _, arg := range p.Prompt.Arguments {
		if arg.Name == req.Params.Argument.Name {
			if p.completer == nil {
				return &mcp.CompleteResult{
					Completion: mcp.CompleteResultCompletion{
						HasMore: utils.Ptr(false),
						Total:   utils.Ptr(0),
						Values:  []string{},
					},
				}, nil
			}

			return p.completer(&arg, req.Params.Argument.Value)
		}
	}

	return nil, fmt.Errorf("%w: '%s'", ErrNoSuchArgument, req.Params.Argument.Name)
}
