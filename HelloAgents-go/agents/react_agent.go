package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// ReActAgent implements the ReAct (Reasoning + Acting) paradigm.
// It follows a thought-action-observation loop for problem solving.
type ReActAgent struct {
	*core.BaseAgent
	toolRegistry *tools.ToolRegistry
	maxSteps     int
	customPrompt string
}

// ReActStep represents a single step in the ReAct loop.
type ReActStep struct {
	Thought     string
	Action      string
	Observation string
}

// NewReActAgent creates a new ReActAgent.
func NewReActAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
) *ReActAgent {
	return &ReActAgent{
		BaseAgent:    core.NewBaseAgent(name, llm, systemPrompt, nil),
		toolRegistry: toolRegistry,
		maxSteps:     10,
		customPrompt: "",
	}
}

// Run executes the ReAct agent with the given input.
func (a *ReActAgent) Run(ctx context.Context, inputText string) (string, error) {
	// Build the initial prompt
	history := make([]ReActStep, 0)

	// Iterate through the ReAct loop
	for step := 0; step < a.maxSteps; step++ {
		// Build prompt with history
		prompt := a.buildPrompt(inputText, history)

		// Call LLM
		messages := []core.ChatMessage{
			{Role: core.RoleSystem, Content: a.SystemPrompt},
			{Role: core.RoleUser, Content: prompt},
		}

		response, err := a.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM invocation failed at step %d: %w", step, err)
		}

		// Parse response into thought and action
		thought, action := a.parseResponse(response)

		// Add to history
		currentStep := ReActStep{Thought: thought, Action: action}
		history = append(history, currentStep)

		// If no action, we're done
		if action == "" || strings.ToLower(action) == "finish" || strings.ToLower(action) == "done" {
			// Extract final answer from thought
			return a.extractFinalAnswer(thought), nil
		}

		// Execute action
		observation, err := a.executeAction(action)
		if err != nil {
			// Add error as observation
			history[len(history)-1].Observation = fmt.Sprintf("Error: %v", err)
		} else {
			history[len(history)-1].Observation = observation
		}
	}

	// Max steps reached - return best answer from last thought
	return a.extractFinalAnswer(history[len(history)-1].Thought), nil
}

// buildPrompt constructs the prompt with current history.
func (a *ReActAgent) buildPrompt(inputText string, history []ReActStep) string {
	var builder strings.Builder

	// Add custom prompt if provided
	if a.customPrompt != "" {
		builder.WriteString(a.customPrompt)
		builder.WriteString("\n\n")
	}

	// Add available tools
	if a.toolRegistry != nil && a.toolRegistry.Count() > 0 {
		builder.WriteString("Available tools:\n")
		for _, tool := range a.toolRegistry.ListTools() {
			builder.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		builder.WriteString("\n")
	}

	// Add input
	builder.WriteString("Question: ")
	builder.WriteString(inputText)
	builder.WriteString("\n\n")

	// Add history
	for i, step := range history {
		if step.Thought != "" {
			builder.WriteString(fmt.Sprintf("Thought %d: %s\n", i+1, step.Thought))
		}
		if step.Action != "" {
			builder.WriteString(fmt.Sprintf("Action %d: %s\n", i+1, step.Action))
		}
		if step.Observation != "" {
			builder.WriteString(fmt.Sprintf("Observation %d: %s\n", i+1, step.Observation))
		}
		builder.WriteString("\n")
	}

	// Add next step prompt
	builder.WriteString("Please provide your next thought and action in the format:")
	builder.WriteString("\nThought: [your reasoning]")
	builder.WriteString("\nAction: [action to take or 'finish' if done]")

	return builder.String()
}

// parseResponse extracts thought and action from the LLM response.
func (a *ReActAgent) parseResponse(response string) (thought, action string) {
	// Try to parse structured format
	thoughtRe := regexp.MustCompile(`(?i)thought\s*[:：]\s*(.+?)(?:\n|$)`)
	actionRe := regexp.MustCompile(`(?i)action\s*[:：]\s*(.+?)(?:\n|$)`)

	thoughtMatches := thoughtRe.FindStringSubmatch(response)
	actionMatches := actionRe.FindStringSubmatch(response)

	if len(thoughtMatches) > 1 {
		thought = strings.TrimSpace(thoughtMatches[1])
	} else {
		// If no explicit thought marker, use the whole response as thought
		thought = strings.TrimSpace(response)
	}

	if len(actionMatches) > 1 {
		action = strings.TrimSpace(actionMatches[1])
	} else {
		// Try to extract action from common patterns
		action = a.extractAction(response)
	}

	return thought, action
}

// extractAction attempts to extract an action from unstructured text.
func (a *ReActAgent) extractAction(response string) string {
	response = strings.ToLower(response)

	// Check for finish/done keywords
	if strings.Contains(response, "finish") || strings.Contains(response, "done") ||
		strings.Contains(response, "complete") || strings.Contains(response, "answer:") {
		return "finish"
	}

	// Try to find tool call pattern
	if strings.Contains(response, "[tool_call:") {
		re := regexp.MustCompile(`\[tool_call:([^\]:]+):([^\]]+)\]`)
		matches := re.FindStringSubmatch(response)
		if len(matches) >= 3 {
			return fmt.Sprintf("Use tool %s with parameters: %s", matches[1], matches[2])
		}
	}

	// Try to find calculator pattern
	if strings.Contains(response, "calculate") || strings.Contains(response, "compute") {
		re := regexp.MustCompile(`(?:calculate|compute)\s*:?\s*([0-9+\-*/^() .%a-zA-Z]+)`)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 1 {
			return fmt.Sprintf("calculator: %s", strings.TrimSpace(matches[1]))
		}
	}

	// Try to find terminal command pattern
	if strings.Contains(response, "terminal") || strings.Contains(response, "command") ||
		strings.Contains(response, "run") || strings.Contains(response, "execute") {
		re := regexp.MustCompile(`(?:terminal|command|run|execute)\s*:?\s*(.+)`)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 1 {
			return fmt.Sprintf("terminal: %s", strings.TrimSpace(matches[1]))
		}
	}

	return ""
}

// executeAction executes an action and returns the observation.
func (a *ReActAgent) executeAction(action string) (string, error) {
	if a.toolRegistry == nil {
		return "", fmt.Errorf("no tool registry available")
	}

	// Parse action
	action = strings.TrimSpace(action)

	// Check for finish action
	if strings.ToLower(action) == "finish" || strings.ToLower(action) == "done" {
		return "", nil
	}

	// Parse tool call format: "tool_name: parameters" or "Use tool tool_name with parameters: ..."
	if strings.Contains(action, ":") {
		parts := strings.SplitN(action, ":", 2)
		if len(parts) == 2 {
			toolName := strings.TrimSpace(parts[0])
			paramsStr := strings.TrimSpace(parts[1])

			// Remove common prefixes
			toolName = strings.TrimPrefix(toolName, "Use tool ")
			toolName = strings.TrimPrefix(toolName, "calculator ")
			toolName = strings.TrimPrefix(toolName, "terminal ")

			// Prepare parameters
			var params map[string]interface{}
			if toolName == "calculator" {
				params = map[string]interface{}{"expression": paramsStr}
			} else if toolName == "terminal" {
				params = map[string]interface{}{"command": paramsStr}
			} else {
				// Try to parse as JSON
				var err error
				params, err = tools.ConvertParameters(paramsStr)
				if err != nil {
					params = map[string]interface{}{"input": paramsStr}
				}
			}

			// Execute tool
			result, err := a.toolRegistry.ExecuteTool(toolName, params)
			if err != nil {
				return "", fmt.Errorf("tool execution failed: %w", err)
			}

			return result, nil
		}
	}

	return "", fmt.Errorf("unable to parse action: %s", action)
}

// extractFinalAnswer extracts the final answer from a thought.
func (a *ReActAgent) extractFinalAnswer(thought string) string {
	// Look for answer patterns
	re := regexp.MustCompile(`(?i)(?:answer|final|result|conclusion)\s*[:：]\s*(.+)`)
	matches := re.FindStringSubmatch(thought)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Return the thought as is
	return strings.TrimSpace(thought)
}

// SetMaxSteps sets the maximum number of reasoning steps.
func (a *ReActAgent) SetMaxSteps(maxSteps int) {
	a.maxSteps = maxSteps
}

// GetMaxSteps returns the maximum number of reasoning steps.
func (a *ReActAgent) GetMaxSteps() int {
	return a.maxSteps
}

// SetCustomPrompt sets a custom prompt template.
func (a *ReActAgent) SetCustomPrompt(prompt string) {
	a.customPrompt = prompt
}

// GetCustomPrompt returns the custom prompt template.
func (a *ReActAgent) GetCustomPrompt() string {
	return a.customPrompt
}

// AddTool registers a new tool with the agent.
func (a *ReActAgent) AddTool(tool tools.Tool, autoExpand bool) error {
	if a.toolRegistry == nil {
		return fmt.Errorf("tool registry not initialized")
	}

	return a.toolRegistry.RegisterTool(tool, autoExpand)
}

// ListTools returns a list of all registered tool names.
func (a *ReActAgent) ListTools() []string {
	if a.toolRegistry == nil {
		return []string{}
	}

	return a.toolRegistry.ListTools()
}
