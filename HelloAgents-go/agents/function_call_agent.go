package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// FunctionCallAgent uses OpenAI's native function calling mechanism.
// It automatically builds JSON schemas for tools and handles multi-round iterations.
type FunctionCallAgent struct {
	*core.BaseAgent
	toolRegistry      *tools.ToolRegistry
	enableToolCalling bool
	defaultToolChoice string // "auto", "none", or specific tool name
	maxToolIterations int
}

// NewFunctionCallAgent creates a new FunctionCallAgent.
func NewFunctionCallAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
) *FunctionCallAgent {
	enableToolCalling := toolRegistry != nil && len(toolRegistry.ListTools()) > 0
	return &FunctionCallAgent{
		BaseAgent:          core.NewBaseAgent(name, llm, systemPrompt, nil),
		toolRegistry:       toolRegistry,
		enableToolCalling:  enableToolCalling,
		defaultToolChoice:  "auto",
		maxToolIterations:  10,
	}
}

// Run executes the agent with the given input.
func (a *FunctionCallAgent) Run(ctx context.Context, inputText string) (string, error) {
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

	// Build tool schemas if tool calling is enabled
	var toolSchemas []map[string]interface{}
	if a.enableToolCalling && len(a.toolRegistry.ListTools()) > 0 {
		toolSchemas = a.buildToolSchemas()
	}

	// Iterate through tool calls
	iterations := 0
	for iterations < a.maxToolIterations {
		iterations++

		// Call LLM with tools
		response, toolCalls, err := a.callWithTools(ctx, messages, toolSchemas)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		// Add assistant response to messages
		messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: response})

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			// Add to history
			a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
			a.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))
			return response, nil
		}

		// Process tool calls
		for _, toolCall := range toolCalls {
			// Execute tool
			result, err := a.executeToolCall(toolCall)
			if err != nil {
				// Add error as tool result
				messages = append(messages, core.ChatMessage{
					Role:    core.RoleTool,
					Content: fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			// Add tool result to messages
			messages = append(messages, core.ChatMessage{
				Role:    core.RoleTool,
				Content: result,
			})
		}
	}

	// Max iterations reached - get final response without tools
	finalResponse, err := a.LLM.Invoke(ctx, messages, &core.InvokeOptions{})
	if err != nil {
		return "", fmt.Errorf("final LLM call failed: %w", err)
	}

	// Add to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalResponse, core.RoleAssistant, core.Time{}, nil))

	return finalResponse, nil
}

// callWithTools calls the LLM with tool schemas and returns the response and any tool calls.
func (a *FunctionCallAgent) callWithTools(
	ctx context.Context,
	messages []core.ChatMessage,
	toolSchemas []map[string]interface{},
) (string, []ToolCallData, error) {
	// For now, we'll use a simplified approach that calls Invoke and parses tool calls
	// In a full implementation, you would use the OpenAI client's function calling API

	// Call LLM
	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", nil, err
	}

	// Parse tool calls from response
	// Since go-openai doesn't directly expose function calling in the simple API,
	// we'll look for JSON-formatted tool calls in the response
	toolCalls := a.parseToolCalls(response)

	return response, toolCalls, nil
}

// parseToolCalls parses tool calls from the LLM response.
// This is a simplified implementation - in production you'd use the actual OpenAI function calling API.
func (a *FunctionCallAgent) parseToolCalls(response string) []ToolCallData {
	// Try to parse as JSON array
	var toolCalls []ToolCallData
	if err := json.Unmarshal([]byte(response), &toolCalls); err == nil {
		return toolCalls
	}

	// If not a JSON array, try to find JSON objects in the text
	// This is a fallback for when the model doesn't return proper JSON
	return []ToolCallData{}
}

// executeToolCall executes a single tool call.
func (a *FunctionCallAgent) executeToolCall(toolCall ToolCallData) (string, error) {
	if a.toolRegistry == nil {
		return "", fmt.Errorf("tool registry not initialized")
	}

	// Get tool
	tool := a.toolRegistry.GetTool(toolCall.Name)
	if tool == nil {
		return "", fmt.Errorf("tool '%s' not found", toolCall.Name)
	}

	// Parse parameters
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Arguments), &params); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// Execute tool
	result, err := tool.Run(params)
	if err != nil {
		return "", err
	}

	return result, nil
}

// buildToolSchemas builds OpenAI function calling schemas for all registered tools.
func (a *FunctionCallAgent) buildToolSchemas() []map[string]interface{} {
	if a.toolRegistry == nil {
		return []map[string]interface{}{}
	}

	toolList := a.toolRegistry.GetAllTools()
	schemas := make([]map[string]interface{}, 0, len(toolList))

	for _, tool := range toolList {
		schemas = append(schemas, tool.ToOpenAISchema())
	}

	return schemas
}

// AddTool registers a new tool with the agent.
func (a *FunctionCallAgent) AddTool(tool tools.Tool, autoExpand bool) {
	if a.toolRegistry == nil {
		a.toolRegistry = tools.NewToolRegistry()
	}

	a.toolRegistry.RegisterTool(tool, autoExpand)

	// Update tool calling enabled status
	a.enableToolCalling = len(a.toolRegistry.ListTools()) > 0
}

// RemoveTool unregisters a tool from the agent.
func (a *FunctionCallAgent) RemoveTool(name string) {
	if a.toolRegistry != nil {
		a.toolRegistry.Unregister(name)
		// Update tool calling enabled status
		a.enableToolCalling = len(a.toolRegistry.ListTools()) > 0
	}
}

// ListTools returns a list of all registered tool names.
func (a *FunctionCallAgent) ListTools() []string {
	if a.toolRegistry == nil {
		return []string{}
	}

	return a.toolRegistry.ListTools()
}

// SetDefaultToolChoice sets the default tool choice for function calls.
// Valid values: "auto", "none", or a specific tool name.
func (a *FunctionCallAgent) SetDefaultToolChoice(choice string) {
	a.defaultToolChoice = choice
}

// GetDefaultToolChoice returns the default tool choice.
func (a *FunctionCallAgent) GetDefaultToolChoice() string {
	return a.defaultToolChoice
}

// SetMaxToolIterations sets the maximum number of tool call iterations.
func (a *FunctionCallAgent) SetMaxToolIterations(max int) {
	a.maxToolIterations = max
}

// GetMaxToolIterations returns the maximum number of tool call iterations.
func (a *FunctionCallAgent) GetMaxToolIterations() int {
	return a.maxToolIterations
}

// EnableToolCalling enables or disables tool calling.
func (a *FunctionCallAgent) EnableToolCalling(enabled bool) {
	a.enableToolCalling = enabled && a.toolRegistry != nil && len(a.toolRegistry.ListTools()) > 0
}

// IsToolCallingEnabled returns whether tool calling is enabled.
func (a *FunctionCallAgent) IsToolCallingEnabled() bool {
	return a.enableToolCalling
}

// ToolCallData represents a function call from the OpenAI API.
type ToolCallData struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Function  FunctionData `json:"function"`
}

// FunctionData represents the function call data.
type FunctionData struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
