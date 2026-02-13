package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// SimpleAgent is a basic conversational agent with optional tool calling support.
// It uses a custom tool call format: [TOOL_CALL:tool_name:parameters]
// 继承 HelloAgentsLLM 及其他基础字段，与 Python 版本保持一致
type SimpleAgent struct {
	// 基础字段（与 Python Agent 基类对应）
	Name         string
	LLM          *core.HelloAgentsLLM
	SystemPrompt string
	Config       *core.Config
	history      []core.Message

	// SimpleAgent 特有字段
	toolRegistry      *tools.ToolRegistry
	enableToolCalling bool
}

// NewSimpleAgent creates a new SimpleAgent.
func NewSimpleAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
) *SimpleAgent {
	return &SimpleAgent{
		Name:              name,
		LLM:               llm,
		SystemPrompt:      systemPrompt,
		Config:            core.DefaultConfig(),
		history:           make([]core.Message, 0),
		toolRegistry:      toolRegistry,
		enableToolCalling: toolRegistry != nil,
	}
}

// AddMessage 添加消息到历史记录，与 Python 的 add_message 对应。
func (a *SimpleAgent) AddMessage(m core.Message) {
	a.history = append(a.history, m)
}

// ClearHistory 清空历史记录，与 Python 的 clear_history 对应。
func (a *SimpleAgent) ClearHistory() {
	a.history = a.history[:0]
}

// GetHistory 返回历史记录的副本，与 Python 的 get_history 对应。
func (a *SimpleAgent) GetHistory() []core.Message {
	out := make([]core.Message, len(a.history))
	copy(out, a.history)
	return out
}

// String 实现 fmt.Stringer，与 Python 的 __str__ 对应。
func (a *SimpleAgent) String() string {
	if a.LLM != nil {
		return fmt.Sprintf("Agent(name=%s, provider=%s)", a.Name, a.LLM.Provider)
	}
	return fmt.Sprintf("Agent(name=%s, provider=)", a.Name)
}

// Run executes the agent with the given input.
func (a *SimpleAgent) Run(ctx context.Context, inputText string) (string, error) {
	// Build messages
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
	}

	// Add history
	for _, msg := range a.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}

	// Add current input
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	// If tool calling is enabled, add tool descriptions to system prompt
	if a.enableToolCalling && a.toolRegistry.Count() > 0 {
		toolDesc := a.toolRegistry.GetToolsDescription()
		messages[0] = core.ChatMessage{
			Role:    core.RoleSystem,
			Content: a.SystemPrompt + "\n\n" + toolDesc + "\n\nWhen you need to use a tool, use the format: [TOOL_CALL:tool_name:parameters]",
		}
	}

	// Invoke LLM
	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("LLM invocation failed: %w", err)
	}

	// Process tool calls if enabled
	if a.enableToolCalling {
		response, err = a.processToolCalls(response)
		if err != nil {
			return "", fmt.Errorf("tool call processing failed: %w", err)
		}
	}

	// Add messages to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))

	return response, nil
}

// processToolCalls processes any tool calls in the response.
func (a *SimpleAgent) processToolCalls(response string) (string, error) {
	maxIterations := 10 // Prevent infinite loops
	iterations := 0

	for iterations < maxIterations {
		iterations++

		// Check if there's a tool call
		toolName, paramsStr, found := tools.ParseToolCall(response)
		if !found {
			break
		}

		// Parse parameters
		var params map[string]interface{}
		var err error

		// Try to parse as JSON first
		if strings.HasPrefix(paramsStr, "{") || strings.HasPrefix(paramsStr, "[") {
			params, err = tools.ConvertParameters(paramsStr)
		} else {
			// Treat as a simple string parameter
			params = map[string]interface{}{
				"input": paramsStr,
			}
		}

		if err != nil {
			return "", fmt.Errorf("failed to parse tool parameters: %w", err)
		}

		// Execute tool
		result, err := a.toolRegistry.ExecuteTool(toolName, params)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Prepare messages for follow-up
		messages := []core.ChatMessage{
			{Role: core.RoleSystem, Content: a.SystemPrompt},
		}

		// Add history
		for _, msg := range a.GetHistory() {
			messages = append(messages, msg.ToChatMessage())
		}

		// Add assistant's tool call
		messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: response})

		// Add tool result
		messages = append(messages, core.ChatMessage{
			Role:    core.RoleUser,
			Content: fmt.Sprintf("Tool %s returned: %s\n\nPlease continue based on this result.", toolName, result),
		})

		// Get follow-up response
		response, err = a.LLM.Invoke(context.Background(), messages, nil)
		if err != nil {
			return "", fmt.Errorf("follow-up LLM invocation failed: %w", err)
		}
	}

	return response, nil
}

// AddTool registers a new tool with the agent.
func (a *SimpleAgent) AddTool(tool tools.Tool, autoExpand bool) error {
	if a.toolRegistry == nil {
		return fmt.Errorf("tool registry not initialized")
	}

	return a.toolRegistry.RegisterTool(tool, autoExpand)
}

// RemoveTool unregisters a tool from the agent.
func (a *SimpleAgent) RemoveTool(name string) error {
	if a.toolRegistry == nil {
		return fmt.Errorf("tool registry not initialized")
	}

	return a.toolRegistry.UnregisterTool(name)
}

// ListTools returns a list of all registered tool names.
func (a *SimpleAgent) ListTools() []string {
	if a.toolRegistry == nil {
		return []string{}
	}

	return a.toolRegistry.ListTools()
}

// EnableToolCalling enables or disables tool calling.
func (a *SimpleAgent) EnableToolCalling(enabled bool) {
	a.enableToolCalling = enabled
}

// IsToolCallingEnabled returns whether tool calling is enabled.
func (a *SimpleAgent) IsToolCallingEnabled() bool {
	return a.enableToolCalling
}

// ProcessStreamingResponse processes a streaming response and handles tool calls.
// This is useful when using Think() instead of Invoke().
func (a *SimpleAgent) ProcessStreamingResponse(ctx context.Context, inputText string, streamCh <-chan string, errCh <-chan error) (string, error) {
	var response strings.Builder

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case chunk, ok := <-streamCh:
			if !ok {
				// Stream closed
				break
			}
			response.WriteString(chunk)
		case err, ok := <-errCh:
			if !ok {
				// Error channel closed without error
				break
			}
			return "", err
		}
	}

	fullResponse := response.String()

	// Process tool calls if enabled
	if a.enableToolCalling {
		var err error
		fullResponse, err = a.processToolCalls(fullResponse)
		if err != nil {
			return "", fmt.Errorf("tool call processing failed: %w", err)
		}
	}

	// Add to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, time.Time{}, nil))
	a.AddMessage(core.NewMessage(fullResponse, core.RoleAssistant, time.Time{}, nil))

	return fullResponse, nil
}

// ExtractToolCalls extracts all tool calls from a response string.
// Returns a list of (toolName, parameters) tuples.
func (a *SimpleAgent) ExtractToolCalls(response string) []ToolCall {
	re := regexp.MustCompile(`\[TOOL_CALL:([^:]+):([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(response, -1)

	calls := make([]ToolCall, 0, len(matches))
	for _, match := range matches {
		if len(match) == 3 {
			calls = append(calls, ToolCall{
				Name:       match[1],
				Parameters: match[2],
			})
		}
	}

	return calls
}

// ToolCall represents a single tool call.
type ToolCall struct {
	Name       string
	Parameters string
}

// HasToolCall checks if a response contains any tool calls.
func (a *SimpleAgent) HasToolCall(response string) bool {
	_, _, found := tools.ParseToolCall(response)
	return found
}

// StripToolCalls removes tool call markers from a response.
func (a *SimpleAgent) StripToolCalls(response string) string {
	re := regexp.MustCompile(`\[TOOL_CALL:[^\]]+\]`)
	return re.ReplaceAllString(response, "")
}
