package fxctx

import (
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"go.uber.org/fx"
)

type Tool interface {
	GetName() string
	GetDescription() *string
	GetInputSchema() mcp.ToolInputSchema
	Callback(args map[string]interface{}) ToolResponse
}

type tool struct {
	name        string
	description string
	inputSchema mcp.ToolInputSchema
	callback    func(args map[string]interface{}) ToolResponse
}

func (t *tool) GetName() string {
	return t.name
}

func (t *tool) GetDescription() *string {
	return &t.description
}

func (t *tool) GetInputSchema() mcp.ToolInputSchema {
	return t.inputSchema
}

func (t *tool) Callback(args map[string]interface{}) ToolResponse {
	return t.callback(args)
}

type ToolResponse struct {
	Meta    map[string]interface{}
	Content []interface{}
	IsError *bool
}

func NewTool(
	name string,
	description string,
	inputSchema mcp.ToolInputSchema,
	callback func(args map[string]interface{}) ToolResponse) Tool {
	return &tool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
		callback:    callback,
	}
}

func AsTool(f any) any {
	return fx.Annotate(f, fx.As(new(Tool)), fx.ResultTags(`group:"tools"`))
}
