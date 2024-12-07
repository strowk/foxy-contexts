package fxctx

import (
	"sort"

	"github.com/strowk/foxy-contexts/internal/jsonrpc2"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/server"
	"go.uber.org/fx"
)

type ToolMux interface {
	GetMcpTools() []mcp.Tool
	CallToolNamed(name string, args map[string]interface{}) ToolResponse
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
		m[tool.GetName()] = tool
	}

	return &toolMux{
		tools: m,
	}
}

func (t *toolMux) CallToolNamed(name string, args map[string]interface{}) ToolResponse {
	tool, ok := t.tools[name]
	if !ok {
		return ToolResponse{}
	}

	return tool.Callback(args)
}

func (t *toolMux) GetMcpTools() []mcp.Tool {
	tools := []mcp.Tool{}

	for _, tool := range t.tools {
		tools = append(tools,
			mcp.Tool{
				Name:        tool.GetName(),
				Description: tool.GetDescription(),
				InputSchema: tool.GetInputSchema(),
			},
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
		res := t.CallToolNamed(toolName, req.Params.Arguments)
		return &mcp.CallToolResult{
			Meta:    res.Meta,
			Content: res.Content,
			IsError: res.IsError,
		}, nil
	})
}
