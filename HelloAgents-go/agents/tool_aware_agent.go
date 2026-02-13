package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// ToolAwareSimpleAgent extends SimpleAgent with enhanced tool call monitoring and logging.
// It provides better visibility into tool usage and can handle streaming tool calls.
type ToolAwareSimpleAgent struct {
	*SimpleAgent
	toolCalls       []ToolCallRecord
	mu              sync.RWMutex
	enableLogging   bool
	detailedLogging bool
}

// ToolCallRecord represents a record of a tool call.
type ToolCallRecord struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     string                 `json:"result"`
	Error      error                  `json:"error,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

// NewToolAwareSimpleAgent creates a new ToolAwareSimpleAgent.
func NewToolAwareSimpleAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
) *ToolAwareSimpleAgent {
	return &ToolAwareSimpleAgent{
		SimpleAgent:     NewSimpleAgent(name, llm, systemPrompt, toolRegistry),
		toolCalls:       make([]ToolCallRecord, 0),
		enableLogging:   true,
		detailedLogging: false,
	}
}

// Run executes the agent with enhanced tool call tracking.
func (a *ToolAwareSimpleAgent) Run(ctx context.Context, inputText string) (string, error) {
	// Clear previous tool call records for this run
	a.ClearToolCallRecords()

	// Run the base agent
	response, err := a.SimpleAgent.Run(ctx, inputText)
	if err != nil {
		return "", err
	}

	// Log tool calls if enabled
	if a.enableLogging {
		a.logToolCalls()
	}

	return response, nil
}

// ProcessStreamingResponse processes a streaming response with tool call detection.
func (a *ToolAwareSimpleAgent) ProcessStreamingResponse(
	ctx context.Context,
	inputText string,
	streamCh <-chan string,
	errCh <-chan error,
) (string, error) {
	var response strings.Builder
	var currentToolCall strings.Builder
	inToolCall := false

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

			// Detect tool call markers in streaming output
			if a.enableLogging {
				chunk = a.detectToolCalls(chunk, &currentToolCall, &inToolCall)
			}
		case err, ok := <-errCh:
			if !ok {
				// Error channel closed without error
				break
			}
			return "", err
		}
	}

	fullResponse := response.String()

	// Process any detected tool calls
	if a.enableLogging && inToolCall {
		a.processStreamingToolCall(currentToolCall.String())
	}

	// Process tool calls if enabled
	if a.IsToolCallingEnabled() {
		var err error
		fullResponse, err = a.processToolCallsWithTracking(fullResponse)
		if err != nil {
			return "", fmt.Errorf("tool call processing failed: %w", err)
		}
	}

	// Add to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(fullResponse, core.RoleAssistant, core.Time{}, nil))

	return fullResponse, nil
}

// detectToolCalls detects tool call patterns in streaming chunks.
func (a *ToolAwareSimpleAgent) detectToolCalls(
	chunk string,
	currentToolCall *strings.Builder,
	inToolCall *bool,
) string {
	// Check for tool call start
	if strings.Contains(chunk, "[TOOL_CALL:") {
		*inToolCall = true
	}

	// Accumulate tool call content
	if *inToolCall {
		currentToolCall.WriteString(chunk)

		// Check for tool call end
		if strings.Contains(chunk, "]") {
			*inToolCall = false
			// Process the accumulated tool call
			a.processStreamingToolCall(currentToolCall.String())
			currentToolCall.Reset()
		}
	}

	return chunk
}

// processStreamingToolCall processes a tool call detected during streaming.
func (a *ToolAwareSimpleAgent) processStreamingToolCall(toolCallStr string) {
	toolName, paramsStr, found := tools.ParseToolCall(toolCallStr)
	if !found {
		return
	}

	// Parse parameters
	params, err := tools.ConvertParameters(paramsStr)
	if err != nil {
		params = map[string]interface{}{"input": paramsStr}
	}

	// Create a pending tool call record
	record := ToolCallRecord{
		ToolName:   toolName,
		Parameters: params,
		Timestamp:  "streaming",
	}

	a.mu.Lock()
	a.toolCalls = append(a.toolCalls, record)
	a.mu.Unlock()
}

// processToolCallsWithTracking processes tool calls and tracks them.
func (a *ToolAwareSimpleAgent) processToolCallsWithTracking(response string) (string, error) {
	maxIterations := 10
	iterations := 0

	currentResponse := response

	for iterations < maxIterations {
		iterations++

		// Check if there's a tool call
		toolName, paramsStr, found := tools.ParseToolCall(currentResponse)
		if !found {
			break
		}

		// Parse parameters with enhanced error handling
		var params map[string]interface{}
		var err error

		if strings.HasPrefix(paramsStr, "{") || strings.HasPrefix(paramsStr, "[") {
			params, err = tools.ConvertParameters(paramsStr)
		} else {
			params = map[string]interface{}{
				"input": paramsStr,
			}
		}

		if err != nil {
			// Record the error
			a.recordToolCall(toolName, params, "", err)
			return "", fmt.Errorf("failed to parse tool parameters: %w", err)
		}

		// Execute tool with error handling
		result, err := a.toolRegistry.ExecuteTool(toolName, params)

		// Record the tool call
		a.recordToolCall(toolName, params, result, err)

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
		messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: currentResponse})

		// Add tool result with enhanced context
		toolResultMsg := fmt.Sprintf("Tool %s returned: %s\n\nPlease continue based on this result.", toolName, result)
		if a.detailedLogging {
			toolResultMsg = fmt.Sprintf("Tool Call Details:\nTool: %s\nParameters: %v\nResult: %s\n\nPlease continue based on this result.",
				toolName, params, result)
		}
		messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: toolResultMsg})

		// Get follow-up response
		currentResponse, err = a.LLM.Invoke(context.Background(), messages, nil)
		if err != nil {
			return "", fmt.Errorf("follow-up LLM invocation failed: %w", err)
		}
	}

	return currentResponse, nil
}

// recordToolCall records a tool call in the history.
func (a *ToolAwareSimpleAgent) recordToolCall(
	toolName string,
	parameters map[string]interface{},
	result string,
	err error,
) {
	record := ToolCallRecord{
		ToolName:   toolName,
		Parameters: parameters,
		Result:     result,
		Error:      err,
		Timestamp:  "recorded",
	}

	a.mu.Lock()
	a.toolCalls = append(a.toolCalls, record)
	a.mu.Unlock()
}

// GetToolCallRecords returns all tool call records.
func (a *ToolAwareSimpleAgent) GetToolCallRecords() []ToolCallRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()

	records := make([]ToolCallRecord, len(a.toolCalls))
	copy(records, a.toolCalls)
	return records
}

// GetToolCallCount returns the number of tool calls made.
func (a *ToolAwareSimpleAgent) GetToolCallCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.toolCalls)
}

// ClearToolCallRecords clears all tool call records.
func (a *ToolAwareSimpleAgent) ClearToolCallRecords() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.toolCalls = make([]ToolCallRecord, 0)
}

// GetToolCallSummary returns a summary of tool calls.
func (a *ToolAwareSimpleAgent) GetToolCallSummary() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	summary := make(map[string]int)
	for _, record := range a.toolCalls {
		summary[record.ToolName]++
	}

	return summary
}

// EnableLogging enables or disables tool call logging.
func (a *ToolAwareSimpleAgent) EnableLogging(enabled bool) {
	a.enableLogging = enabled
}

// IsLoggingEnabled returns whether logging is enabled.
func (a *ToolAwareSimpleAgent) IsLoggingEnabled() bool {
	return a.enableLogging
}

// EnableDetailedLogging enables or disables detailed parameter/result logging.
func (a *ToolAwareSimpleAgent) EnableDetailedLogging(enabled bool) {
	a.detailedLogging = enabled
}

// IsDetailedLoggingEnabled returns whether detailed logging is enabled.
func (a *ToolAwareSimpleAgent) IsDetailedLoggingEnabled() bool {
	return a.detailedLogging
}

// logToolCalls logs tool call information.
func (a *ToolAwareSimpleAgent) logToolCalls() {
	if !a.enableLogging {
		return
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.toolCalls) == 0 {
		return
	}

	// Log summary
	summary := a.GetToolCallSummary()
	fmt.Printf("Tool Call Summary: %v\n", summary)

	// Log detailed records if enabled
	if a.detailedLogging {
		for i, record := range a.toolCalls {
			fmt.Printf("Tool Call %d: %s\n", i+1, record.ToolName)
			fmt.Printf("  Parameters: %v\n", record.Parameters)
			if record.Error != nil {
				fmt.Printf("  Error: %v\n", record.Error)
			} else {
				fmt.Printf("  Result: %s\n", record.Result)
			}
		}
	}
}
