package core

import "time"

type EventType string

const (
	AgentStart  EventType = "agent_start"
	AgentFinish EventType = "agent_finish"
	AgentError  EventType = "agent_error"

	StepStart  EventType = "step_start"
	StepFinish EventType = "step_finish"

	LLMStart  EventType = "llm_start"
	LLMChunk  EventType = "llm_chunk"
	LLMFinish EventType = "llm_finish"

	ToolCall   EventType = "tool_call"
	ToolResult EventType = "tool_result"
	ToolError  EventType = "tool_error"

	Thinking   EventType = "thinking"
	Reflection EventType = "reflection"
	Plan       EventType = "plan"
)

// AgentEvent represents lifecycle events for sync/async pipelines.
type AgentEvent struct {
	Type      EventType      `json:"type"`
	Timestamp float64        `json:"timestamp"`
	AgentName string         `json:"agent_name"`
	Data      map[string]any `json:"data"`
}

func NewAgentEvent(eventType EventType, agentName string, data map[string]any) AgentEvent {
	if data == nil {
		data = map[string]any{}
	}
	return AgentEvent{
		Type:      eventType,
		Timestamp: float64(time.Now().UnixNano()) / 1e9,
		AgentName: agentName,
		Data:      data,
	}
}

func (e AgentEvent) ToMap() map[string]any {
	return map[string]any{
		"type":       string(e.Type),
		"timestamp":  e.Timestamp,
		"agent_name": e.AgentName,
		"data":       e.Data,
	}
}

// ToDict keeps naming parity with Python AgentEvent.to_dict().
func (e AgentEvent) ToDict() map[string]any {
	return e.ToMap()
}

// LifecycleHook is intentionally simple for scaffold stage.
type LifecycleHook func(AgentEvent) error

// ExecutionContext mirrors Python execution context container.
type ExecutionContext struct {
	InputText   string         `json:"input_text"`
	CurrentStep int            `json:"current_step"`
	TotalTokens int            `json:"total_tokens"`
	Metadata    map[string]any `json:"metadata"`
}

func NewExecutionContext(inputText string) ExecutionContext {
	return ExecutionContext{
		InputText:   inputText,
		CurrentStep: 0,
		TotalTokens: 0,
		Metadata:    map[string]any{},
	}
}

func (c *ExecutionContext) IncrementStep() {
	c.CurrentStep++
}

func (c *ExecutionContext) AddTokens(tokens int) {
	c.TotalTokens += tokens
}

func (c *ExecutionContext) SetMetadata(key string, value any) {
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	c.Metadata[key] = value
}

func (c *ExecutionContext) GetMetadata(key string, defaultValue any) any {
	if c.Metadata == nil {
		return defaultValue
	}
	v, ok := c.Metadata[key]
	if !ok {
		return defaultValue
	}
	return v
}
