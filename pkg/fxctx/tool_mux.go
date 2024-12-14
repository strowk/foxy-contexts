package fxctx

import (
	"errors"
	"fmt"
	"sort"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

var (
	ErrToolNotFound = errors.New("tool not found")
)

type ToolMux interface {
	GetMcpTools() []mcp.Tool
	CallToolNamed(name string, args map[string]interface{}) (*mcp.CallToolResult, error)
	RegisterHandlers(s server.Server)
}

type toolMux struct {
	tools map[string]Tool
}

func NewToolMux(
	tools []Tool,
) ToolMux {
	m := map[string]Tool{}

	for _, tool := range tools {
		m[tool.GetMcpTool().Name] = tool
	}

	return &toolMux{
		tools: m,
	}
}

func (t *toolMux) CallToolNamed(name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	tool, ok := t.tools[name]
	if !ok {
		return nil, ErrToolNotFound
	}

	return tool.Callback(args), nil
}

func (t *toolMux) GetMcpTools() []mcp.Tool {
	tools := []mcp.Tool{}

	for _, tool := range t.tools {
		tools = append(tools,
			*tool.GetMcpTool(),
		)
	}

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

func ProvideToolMux() fx.Option {
	return fx.Provide(fx.Annotate(
		NewToolMux,
		fx.ParamTags(`group:"tools"`),
	))
}

func (t *toolMux) RegisterHandlers(s server.Server) {
	t.setToolsListHandler(s)
	t.setCallToolHandler(s)
}

func (t *toolMux) setToolsListHandler(s server.Server) {
	s.SetRequestHandler(&mcp.ListToolsRequest{}, func(r jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		tools := t.GetMcpTools()
		return &mcp.ListToolsResult{
			Tools: tools,
		}, nil
	})
}

func (t *toolMux) setCallToolHandler(s server.Server) {
	s.SetRequestHandler(&mcp.CallToolRequest{}, func(r jsonrpc2.Request) (jsonrpc2.Result, *jsonrpc2.Error) {
		req := r.(*mcp.CallToolRequest)
		toolName := req.Params.Name
		res, err := t.CallToolNamed(toolName, req.Params.Arguments)
		if err != nil {
			return nil, jsonrpc2.NewServerError(ToolNotFound, fmt.Sprintf("tool not found: %s", toolName))
		}

		return &mcp.CallToolResult{
			Meta:    res.Meta,
			Content: res.Content,
			IsError: res.IsError,
		}, nil
	})
}
