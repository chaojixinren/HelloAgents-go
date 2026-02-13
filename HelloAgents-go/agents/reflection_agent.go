package agents

import (
	"context"
	"fmt"

	"helloagents-go/HelloAgents-go/core"
)

// ReflectionAgent implements multi-round reflection and refinement.
// It generates an initial answer, reflects on it, and refines it iteratively.
type ReflectionAgent struct {
	*core.BaseAgent
	maxIterations int
	customPrompts  map[string]string
}

// NewReflectionAgent creates a new ReflectionAgent.
func NewReflectionAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
) *ReflectionAgent {
	return &ReflectionAgent{
		BaseAgent:     core.NewBaseAgent(name, llm, systemPrompt, nil),
		maxIterations: 3,
		customPrompts: make(map[string]string),
	}
}

// Run executes the reflection agent with the given input.
func (a *ReflectionAgent) Run(ctx context.Context, inputText string) (string, error) {
	// Generate initial answer
	result, err := a.generateInitialAnswer(ctx, inputText)
	if err != nil {
		return "", fmt.Errorf("failed to generate initial answer: %w", err)
	}

	// Iteratively reflect and refine
	for i := 0; i < a.maxIterations; i++ {
		// Generate reflection
		reflection, err := a.reflect(ctx, inputText, result)
		if err != nil {
			return "", fmt.Errorf("failed to generate reflection at iteration %d: %w", i+1, err)
		}

		// Refine based on reflection
		result, err = a.refine(ctx, inputText, result, reflection)
		if err != nil {
			return "", fmt.Errorf("failed to refine at iteration %d: %w", i+1, err)
		}
	}

	// Add to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(result, core.RoleAssistant, core.Time{}, nil))

	return result, nil
}

// generateInitialAnswer generates the initial answer.
func (a *ReflectionAgent) generateInitialAnswer(ctx context.Context, inputText string) (string, error) {
	prompt := a.getPrompt("initial", fmt.Sprintf(
		"Please provide a comprehensive response to the following request:\n\n%s\n\n"+
			"Take your time to think through this carefully and provide a well-structured answer.",
		inputText,
	))

	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
		{Role: core.RoleUser, Content: prompt},
	}

	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	return response, nil
}

// reflect generates a reflection on the current result.
func (a *ReflectionAgent) reflect(ctx context.Context, inputText, result string) (string, error) {
	prompt := a.getPrompt("reflect", fmt.Sprintf(
		"Original request:\n%s\n\nCurrent response:\n%s\n\n"+
			"Please critically analyze this response. Consider:\n"+
			"1. What are the strengths of this response?\n"+
			"2. What could be improved?\n"+
			"3. Is there any missing information?\n"+
			"4. Are there any errors or inaccuracies?\n"+
			"5. How could the response be made more clear or comprehensive?\n\n"+
			"Provide specific, constructive feedback.",
		inputText, result,
	))

	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
		{Role: core.RoleUser, Content: prompt},
	}

	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	return response, nil
}

// refine generates an improved version based on reflection.
func (a *ReflectionAgent) refine(ctx context.Context, inputText, result, reflection string) (string, error) {
	prompt := a.getPrompt("refine", fmt.Sprintf(
		"Original request:\n%s\n\nPrevious response:\n%s\n\nFeedback:\n%s\n\n"+
			"Please revise and improve your response based on the feedback above. "+
			"Address all the points raised in the feedback and provide a better, more comprehensive answer. "+
			"Maintain what was good and improve what needs improvement.",
		inputText, result, reflection,
	))

	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
		{Role: core.RoleUser, Content: prompt},
	}

	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	return response, nil
}

// getPrompt returns a custom prompt if set, otherwise returns the default prompt.
func (a *ReflectionAgent) getPrompt(promptType, defaultPrompt string) string {
	if customPrompt, exists := a.customPrompts[promptType]; exists {
		return customPrompt
	}
	return defaultPrompt
}

// SetMaxIterations sets the maximum number of reflection iterations.
func (a *ReflectionAgent) SetMaxIterations(maxIterations int) {
	a.maxIterations = maxIterations
}

// GetMaxIterations returns the maximum number of reflection iterations.
func (a *ReflectionAgent) GetMaxIterations() int {
	return a.maxIterations
}

// SetCustomPrompt sets a custom prompt for a specific type.
// Valid types: "initial", "reflect", "refine"
func (a *ReflectionAgent) SetCustomPrompt(promptType, prompt string) {
	a.customPrompts[promptType] = prompt
}

// GetCustomPrompt returns the custom prompt for a specific type.
func (a *ReflectionAgent) GetCustomPrompt(promptType string) (string, bool) {
	prompt, exists := a.customPrompts[promptType]
	return prompt, exists
}

// ClearCustomPrompts removes all custom prompts.
func (a *ReflectionAgent) ClearCustomPrompts() {
	a.customPrompts = make(map[string]string)
}
