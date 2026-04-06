package acp

// ACP protocol v1 method names (kiro-cli 1.28.x)
const (
	MethodInitialize = "initialize"
	MethodNewSession = "session/new"
	MethodPrompt     = "session/prompt"
	MethodCancel     = "session/cancel"
	NotifUpdate      = "session/update"
	NotifUpdateKiro  = "_kiro.dev/session/update"

	ClientProtocolVersion = "2025-11-16"
)

// session/update notification types (kiro-cli 1.28.2+)
const (
	UpdateToolCallChunk  = "tool_call_chunk"
	UpdateToolCall       = "tool_call"
	UpdateToolCallUpdate = "tool_call_update"
	UpdateAgentChunk     = "agent_message_chunk"
)

// ToolCallEvent carries parsed tool call notification data.
type ToolCallEvent struct {
	ToolCallID string
	Title      string // human-readable, e.g. "Running: echo hello"
	Kind       string // "execute"
	Status     string // "completed" / "failed" (only in tool_call_update)
	RawInput   map[string]interface{}
	RawOutput  string
}

// InitializeResult holds the agent's initialize response.
type InitializeResult struct {
	ProtocolVersion interface{} `json:"protocolVersion"` // numeric 1 or string
	AgentInfo       *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"agentInfo,omitempty"`
}
