package mcp

// This is patched config with the spec that has certain adjustments:
//go:generate go-jsonschema -p mcp https://raw.githubusercontent.com/strowk/mcp-specification/refs/heads/main/schema/schema.json -o ./schema.go

// This is older config with original spec:
//  //go:generate go-jsonschema -p mcp https://raw.githubusercontent.com/modelcontextprotocol/specification/refs/heads/main/schema/schema.json -o ./schema.go

// We use here the go-jsonschema tool to generate most of needed Go types from the JSON schema,
// however methods of types are not included in the schema, so we need to list them here.

func (r ListResourcesRequest) GetMethod() string {
	return "resources/list"
}

func (r ReadResourceRequest) GetMethod() string {
	return "resources/read"
}

func (r ListResourceTemplatesRequest) GetMethod() string {
	return "resources/templates/list"
}

func (r SubscribeRequest) GetMethod() string {
	return "resources/subscribe"
}

func (r UnsubscribeRequest) GetMethod() string {
	return "resources/unsubscribe"
}

func (r InitializeRequest) GetMethod() string {
	return "initialize"
}

func (n InitializedNotification) GetMethod() string {
	return "notifications/initialized"
}

func (r PingRequest) GetMethod() string {
	return "ping"
}

func (r ListToolsRequest) GetMethod() string {
	return "tools/list"
}

func (r CallToolRequest) GetMethod() string {
	return "tools/call"
}

func (r ProgressNotification) GetMethod() string {
	return "notifications/progress"
}

func (r ResourceUpdatedNotification) GetMethod() string {
	return "notifications/resources/updated"
}

func (r ListPromptsRequest) GetMethod() string {
	return "prompts/list"
}

func (r GetPromptRequest) GetMethod() string {
	return "prompts/get"
}

func (r PromptListChangedNotification) GetMethod() string {
	return "notifications/prompts/list_changed"
}

func (r ToolListChangedNotification) GetMethod() string {
	return "notifications/tools/list_changed"
}

func (r LoggingLevel) GetMethod() string {
	return "logging/setLevel"
}

func (r LoggingMessageNotification) GetMethod() string {
	return "notifications/message"
}

func (r CreateMessageRequest) GetMethod() string {
	return "sampling/createMessage"
}

func (r CompleteRequest) GetMethod() string {
	return "completion/complete"
}

func (r ListRootsRequest) GetMethod() string {
	return "roots/list"
}

func (r RootsListChangedNotification) GetMethod() string {
	return "notifications/roots/list_changed"
}

func (r CancelledNotification) GetMethod() string {
	return "notifications/cancelled"
}
