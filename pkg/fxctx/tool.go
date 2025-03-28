package fxctx

import (
	"context"

	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type Tool interface {
	GetMcpTool() *mcp.Tool
	Callback(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult
}

type tool struct {
	mcpTool  *mcp.Tool
	callback func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult
}

func (t *tool) GetMcpTool() *mcp.Tool {
	return t.mcpTool
}

func (t *tool) Callback(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
	return t.callback(ctx, args)
}

type ToolResponse struct {
	Meta    map[string]interface{}
	Content []interface{}
	IsError *bool
}

func NewTool(
	mcpTool *mcp.Tool,
	callback func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult) Tool {
	return &tool{
		mcpTool:  mcpTool,
		callback: callback,
	}
}

func AsTool(f any) any {
	return fx.Annotate(f, fx.As(new(Tool)), fx.ResultTags(`group:"tools"`))
}
