package agents

import (
	"context"
	"fmt"
	"strings"

	"helloagents-go/HelloAgents-go/core"
)

// PlanAndSolveAgent separates planning and execution phases.
// It first creates a plan, then executes each step sequentially.
type PlanAndSolveAgent struct {
	*core.BaseAgent
	customPrompts map[string]string
}

// NewPlanAndSolveAgent creates a new PlanAndSolveAgent.
func NewPlanAndSolveAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
) *PlanAndSolveAgent {
	return &PlanAndSolveAgent{
		BaseAgent:     core.NewBaseAgent(name, llm, systemPrompt, nil),
		customPrompts: make(map[string]string),
	}
}

// Run executes the PlanAndSolve agent with the given input.
func (a *PlanAndSolveAgent) Run(ctx context.Context, inputText string) (string, error) {
	// Phase 1: Planning
	plan, err := a.plan(ctx, inputText)
	if err != nil {
		return "", fmt.Errorf("planning phase failed: %w", err)
	}

	if len(plan) == 0 {
		return "", fmt.Errorf("no plan was generated")
	}

	// Phase 2: Execution
	var result strings.Builder
	var history string

	for i, step := range plan {
		// Execute current step
		stepResult, err := a.execute(ctx, inputText, plan, history, step)
		if err != nil {
			return "", fmt.Errorf("execution failed at step %d: %w", i+1, err)
		}

		// Build history for next step
		if history != "" {
			history += "\n\n"
		}
		history += fmt.Sprintf("Step %d: %s\nResult: %s", i+1, step, stepResult)

		// Accumulate results
		if result.Len() > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(fmt.Sprintf("Step %d (%s):\n%s", i+1, step, stepResult))
	}

	// Phase 3: Synthesis
	finalAnswer, err := a.synthesize(ctx, inputText, plan, history)
	if err != nil {
		// If synthesis fails, return the accumulated results
		return result.String(), nil
	}

	// Add to history
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.RoleAssistant, core.Time{}, nil))

	return finalAnswer, nil
}

// plan generates a step-by-step plan to solve the problem.
func (a *PlanAndSolveAgent) plan(ctx context.Context, inputText string) ([]string, error) {
	prompt := a.getPrompt("plan", fmt.Sprintf(
		"Please analyze the following request and create a detailed step-by-step plan to address it.\n\n"+
			"Request:\n%s\n\n"+
			"Your plan should:\n"+
			"1. Break down the problem into clear, logical steps\n"+
			"2. Number each step (e.g., 'Step 1:', 'Step 2:', etc.)\n"+
			"3. Be specific about what each step should accomplish\n"+
			"4. Ensure the steps are in a logical order\n"+
			"5. Be thorough but concise\n\n"+
			"Format your response as a numbered list of steps.",
		inputText,
	))

	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
		{Role: core.RoleUser, Content: prompt},
	}

	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return nil, err
	}

	// Parse the plan into steps
	steps := a.parsePlan(response)
	return steps, nil
}

// parsePlan extracts individual steps from the plan text.
func (a *PlanAndSolveAgent) parsePlan(planText string) []string {
	lines := strings.Split(planText, "\n")
	steps := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for numbered steps
		if strings.HasPrefix(line, "Step") ||
		   strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") ||
		   strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") ||
		   strings.HasPrefix(line, "5.") || strings.HasPrefix(line, "6.") ||
		   strings.HasPrefix(line, "7.") || strings.HasPrefix(line, "8.") ||
		   strings.HasPrefix(line, "9.") || strings.HasPrefix(line, "10.") {

			// Remove the number/step prefix
			step := strings.TrimSpace(line)
			if idx := strings.Index(step, ":"); idx > 0 {
				step = strings.TrimSpace(step[idx+1:])
			} else {
				// Remove "Step N:" or "N." prefix
				parts := strings.Fields(step)
				if len(parts) > 1 {
					step = strings.Join(parts[1:], " ")
				}
			}

			if step != "" {
				steps = append(steps, step)
			}
		}
	}

	// If no structured steps found, treat each non-empty line as a step
	if len(steps) == 0 {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(strings.ToLower(line), "here") {
				steps = append(steps, line)
			}
		}
	}

	return steps
}

// execute executes a single step with context from previous steps.
func (a *PlanAndSolveAgent) execute(
	ctx context.Context,
	inputText string,
	plan []string,
	history string,
	currentStep string,
) (string, error) {
	prompt := a.getPrompt("execute", fmt.Sprintf(
		"Original request:\n%s\n\nPlan:\n%s\n\nProgress:\n%s\n\n"+
			"Current step to execute:\n%s\n\n"+
			"Please execute this step, taking into account the original request, the overall plan, "+
			"and what has been accomplished so far. Provide a clear and complete result for this step.",
		inputText, strings.Join(plan, "\n"), history, currentStep,
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

// synthesize combines all the results into a final answer.
func (a *PlanAndSolveAgent) synthesize(
	ctx context.Context,
	inputText string,
	plan []string,
	history string,
) (string, error) {
	prompt := a.getPrompt("synthesize", fmt.Sprintf(
		"Original request:\n%s\n\nPlan:\n%s\n\nExecution history:\n%s\n\n"+
			"Based on the plan and its execution, please provide a comprehensive final answer to the original request. "+
			"Synthesize all the information and results into a clear, well-structured response that directly addresses the original request.",
		inputText, strings.Join(plan, "\n"), history,
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
func (a *PlanAndSolveAgent) getPrompt(promptType, defaultPrompt string) string {
	if customPrompt, exists := a.customPrompts[promptType]; exists {
		return customPrompt
	}
	return defaultPrompt
}

// SetCustomPrompt sets a custom prompt for a specific phase.
// Valid types: "plan", "execute", "synthesize"
func (a *PlanAndSolveAgent) SetCustomPrompt(promptType, prompt string) {
	a.customPrompts[promptType] = prompt
}

// GetCustomPrompt returns the custom prompt for a specific phase.
func (a *PlanAndSolveAgent) GetCustomPrompt(promptType string) (string, bool) {
	prompt, exists := a.customPrompts[promptType]
	return prompt, exists
}

// ClearCustomPrompts removes all custom prompts.
func (a *PlanAndSolveAgent) ClearCustomPrompts() {
	a.customPrompts = make(map[string]string)
}
