package agents

import (
	"fmt"
	"strings"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

type Planner struct {
	LLMClient    *core.HelloAgentsLLM
	SystemPrompt string
}

func NewPlanner(llm *core.HelloAgentsLLM, systemPrompt string) *Planner {
	if systemPrompt == "" {
		systemPrompt = `你是一个顶级的AI规划专家。你的任务是将用户提出的复杂问题分解成一个由多个简单步骤组成的行动计划。
请确保计划中的每个步骤都是一个独立的、可执行的子任务，并且严格按照逻辑顺序排列。`
	}
	return &Planner{LLMClient: llm, SystemPrompt: systemPrompt}
}

func (p *Planner) Plan(question string, kwargs map[string]any) ([]string, error) {
	planTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "generate_plan",
			"description": "生成解决问题的分步计划",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"steps": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "按顺序排列的执行步骤列表",
					},
				},
				"required": []string{"steps"},
			},
		},
	}

	messages := []map[string]any{
		{"role": "system", "content": p.SystemPrompt},
		{"role": "user", "content": "请为以下问题生成详细的执行计划：\n\n" + question},
	}

	response, err := p.LLMClient.InvokeWithTools(
		messages,
		[]map[string]any{planTool},
		map[string]any{"type": "function", "function": map[string]any{"name": "generate_plan"}},
		kwargs,
	)
	if err != nil {
		return []string{}, nil
	}

	_, toolCalls := extractToolCallsAndContent(response)
	if len(toolCalls) == 0 {
		return []string{}, nil
	}

	args := toolCalls[0].Arguments
	rawSteps, _ := args["steps"].([]any)
	steps := make([]string, 0, len(rawSteps))
	for _, raw := range rawSteps {
		step := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if step != "" {
			steps = append(steps, step)
		}
	}
	return steps, nil
}

type Executor struct {
	LLMClient         *core.HelloAgentsLLM
	SystemPrompt      string
	ToolRegistry      *tools.ToolRegistry
	EnableToolCalling bool
	MaxToolIterations int
}

func NewExecutor(llm *core.HelloAgentsLLM, systemPrompt string, toolRegistry *tools.ToolRegistry, enableToolCalling bool, maxToolIterations int) *Executor {
	if systemPrompt == "" {
		systemPrompt = `你是一位顶级的AI执行专家。你的任务是严格按照给定的计划，一步步地解决问题。
请专注于解决当前步骤，并输出该步骤的最终答案。`
	}
	if maxToolIterations <= 0 {
		maxToolIterations = 3
	}
	return &Executor{
		LLMClient:         llm,
		SystemPrompt:      systemPrompt,
		ToolRegistry:      toolRegistry,
		EnableToolCalling: enableToolCalling && toolRegistry != nil,
		MaxToolIterations: maxToolIterations,
	}
}

func (e *Executor) Execute(question string, plan []string, kwargs map[string]any) (string, error) {
	history := make([]map[string]string, 0, len(plan))
	finalAnswer := ""

	for _, step := range plan {
		context := fmt.Sprintf(`# 原始问题:
%s

# 完整计划:
%s

# 历史步骤与结果:
%s

# 当前步骤:
%s

请执行当前步骤并给出结果。`,
			question,
			e.formatPlan(plan),
			e.formatHistory(history),
			step,
		)

		responseText, err := e.executeStep(context, kwargs)
		if err != nil {
			return "", err
		}

		history = append(history, map[string]string{"step": step, "result": responseText})
		finalAnswer = responseText
	}

	return finalAnswer, nil
}

func (e *Executor) formatPlan(plan []string) string {
	lines := make([]string, 0, len(plan))
	for i, step := range plan {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
	}
	return strings.Join(lines, "\n")
}

func (e *Executor) formatHistory(history []map[string]string) string {
	if len(history) == 0 {
		return "无"
	}
	parts := make([]string, 0, len(history))
	for i, item := range history {
		parts = append(parts, fmt.Sprintf("步骤 %d: %s\n结果: %s", i+1, item["step"], item["result"]))
	}
	return strings.Join(parts, "\n\n")
}

func (e *Executor) executeStep(context string, kwargs map[string]any) (string, error) {
	messages := []map[string]any{
		{"role": "system", "content": e.SystemPrompt},
		{"role": "user", "content": context},
	}

	if !e.EnableToolCalling || e.ToolRegistry == nil {
		llmResponse, err := e.LLMClient.Invoke(messages, kwargs)
		if err != nil {
			return "", err
		}
		return llmResponse.Content, nil
	}

	tempAgent, err := NewSimpleAgent("temp_executor", e.LLMClient, "", nil, e.ToolRegistry)
	if err != nil {
		return "", err
	}
	tempAgent.EnableToolCalling = true
	toolSchemas := tempAgent.BuildToolSchemas()

	currentIteration := 0
	for currentIteration < e.MaxToolIterations {
		currentIteration++
		response, err := e.LLMClient.InvokeWithTools(messages, toolSchemas, "auto", kwargs)
		if err != nil {
			break
		}

		content, toolCalls := extractToolCallsAndContent(response)
		if len(toolCalls) == 0 {
			return content, nil
		}

		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    content,
			"tool_calls": toOpenAIToolCallsPayload(toolCalls),
		})

		for _, toolCall := range toolCalls {
			if toolCall.ParseError != "" {
				messages = append(messages, map[string]any{
					"role":         "tool",
					"tool_call_id": toolCall.ID,
					"content":      "错误：参数格式不正确 - " + toolCall.ParseError,
				})
				continue
			}

			arguments := toolCall.Arguments
			if arguments == nil {
				arguments = map[string]any{}
			}
			result := tempAgent.ExecuteToolCall(toolCall.Name, arguments)
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      result,
			})
		}
	}

	if currentIteration >= e.MaxToolIterations {
		llmResponse, err := e.LLMClient.Invoke(messages, kwargs)
		if err != nil {
			return "", err
		}
		return llmResponse.Content, nil
	}
	return "", nil
}

// PlanSolveAgent mirrors hello_agents.agents.plan_solve_agent.PlanSolveAgent.
type PlanSolveAgent struct {
	*core.BaseAgent
	Planner  *Planner
	Executor *Executor
}

type PlanAndSolveAgent = PlanSolveAgent

func NewPlanSolveAgent(name string, llm *core.HelloAgentsLLM, systemPrompt string, config *core.Config, toolRegistry *tools.ToolRegistry) (*PlanSolveAgent, error) {
	base, err := core.NewBaseAgent(name, llm, systemPrompt, config, toolRegistry)
	if err != nil {
		return nil, err
	}

	agent := &PlanSolveAgent{
		BaseAgent: base,
		Planner:   NewPlanner(llm, ""),
		Executor:  NewExecutor(llm, "", toolRegistry, true, 3),
	}
	base.AgentType = "PlanSolveAgent"
	autoRegisterBuiltinTools(base)
	return agent, nil
}

func NewPlanAndSolveAgent(name string, llm *core.HelloAgentsLLM, systemPrompt string, config *core.Config, toolRegistry *tools.ToolRegistry) (*PlanSolveAgent, error) {
	return NewPlanSolveAgent(name, llm, systemPrompt, config, toolRegistry)
}

func (a *PlanSolveAgent) Run(inputText string, kwargs map[string]any) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}

	plan, err := a.Planner.Plan(inputText, kwargs)
	if err != nil {
		return "", err
	}
	if len(plan) == 0 {
		finalAnswer := "无法生成有效的行动计划，任务终止。"
		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
		return finalAnswer, nil
	}

	finalAnswer, err := a.Executor.Execute(inputText, plan, kwargs)
	if err != nil {
		return "", err
	}

	a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
	return finalAnswer, nil
}

func (a *PlanSolveAgent) GetToolRegistry() *tools.ToolRegistry {
	return a.ToolRegistry
}

func (a *PlanSolveAgent) String() string {
	return fmt.Sprintf("PlanSolveAgent(name=%s)", a.Name)
}

func (a *PlanSolveAgent) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	return runAsSubagent(a, task, toolFilter, returnSummary, maxStepsOverride)
}
