package agents

import (
	"fmt"
	"strings"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

var defaultReflectionSystemPrompt = `你是一个具有自我反思能力的AI助手。你的工作流程是：
1. 首先尝试完成用户的任务
2. 然后反思你的回答，找出可能的问题或改进空间
3. 根据反思结果优化你的回答
4. 如果回答已经很好，在反思时回复"无需改进"

请始终保持批判性思维，追求更高质量的输出。`

type reflectionRecord struct {
	Type    string
	Content string
}

type reflectionMemory struct {
	records []reflectionRecord
}

func newReflectionMemory() *reflectionMemory {
	return &reflectionMemory{records: []reflectionRecord{}}
}

func (m *reflectionMemory) addRecord(recordType string, content string) {
	m.records = append(m.records, reflectionRecord{Type: recordType, Content: content})
}

func (m *reflectionMemory) getTrajectory() string {
	if len(m.records) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m.records))
	for _, record := range m.records {
		if record.Type == "execution" {
			parts = append(parts, fmt.Sprintf("--- 上一轮尝试 (代码) ---\n%s", record.Content))
			continue
		}
		if record.Type == "reflection" {
			parts = append(parts, fmt.Sprintf("--- 评审员反馈 ---\n%s", record.Content))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func (m *reflectionMemory) getLastExecution() string {
	for i := len(m.records) - 1; i >= 0; i-- {
		if m.records[i].Type == "execution" {
			return m.records[i].Content
		}
	}
	return ""
}

// ReflectionAgent mirrors hello_agents.agents.reflection_agent.ReflectionAgent.
type ReflectionAgent struct {
	*core.BaseAgent
	MaxIterations     int
	Memory            *reflectionMemory
	EnableToolCalling bool
	MaxToolIterations int
}

func NewReflectionAgent(name string, llm *core.HelloAgentsLLM, systemPrompt string, config *core.Config, toolRegistry *tools.ToolRegistry) (*ReflectionAgent, error) {
	return NewReflectionAgentWithOptions(name, llm, systemPrompt, config, 3, toolRegistry, true, 3)
}

func NewReflectionAgentWithOptions(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	config *core.Config,
	maxIterations int,
	toolRegistry *tools.ToolRegistry,
	enableToolCalling bool,
	maxToolIterations int,
) (*ReflectionAgent, error) {
	if systemPrompt == "" {
		systemPrompt = defaultReflectionSystemPrompt
	}
	base, err := core.NewBaseAgent(name, llm, systemPrompt, config, toolRegistry)
	if err != nil {
		return nil, err
	}

	agent := &ReflectionAgent{
		BaseAgent:         base,
		MaxIterations:     maxIterations,
		Memory:            newReflectionMemory(),
		EnableToolCalling: enableToolCalling && toolRegistry != nil,
		MaxToolIterations: maxToolIterations,
	}
	base.SetRunDelegate(agent.Run)
	base.AgentType = "ReflectionAgent"
	autoRegisterBuiltinTools(base)
	return agent, nil
}

func (a *ReflectionAgent) Run(inputText string, kwargs map[string]any) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	a.Memory = newReflectionMemory()

	initialResult, err := a.executeTask(inputText, kwargs)
	if err != nil {
		return "", err
	}
	a.Memory.addRecord("execution", initialResult)

	for i := 0; i < a.MaxIterations; i++ {
		lastResult := a.Memory.getLastExecution()
		feedback, err := a.reflectOnResult(inputText, lastResult, kwargs)
		if err != nil {
			return "", err
		}
		a.Memory.addRecord("reflection", feedback)

		if strings.Contains(feedback, "无需改进") || strings.Contains(strings.ToLower(feedback), "no need for improvement") {
			break
		}

		refinedResult, err := a.refineResult(inputText, lastResult, feedback, kwargs)
		if err != nil {
			return "", err
		}
		a.Memory.addRecord("execution", refinedResult)
	}

	finalResult := a.Memory.getLastExecution()
	a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
	a.AddMessage(core.NewMessage(finalResult, core.MessageRoleAssistant, nil))
	return finalResult, nil
}

func (a *ReflectionAgent) executeTask(task string, kwargs map[string]any) (string, error) {
	messages := []map[string]any{
		{"role": "system", "content": a.SystemPrompt},
		{"role": "user", "content": "请完成以下任务：\n\n" + task},
	}
	return a.getLLMResponse(messages, kwargs)
}

func (a *ReflectionAgent) reflectOnResult(task string, result string, kwargs map[string]any) (string, error) {
	prompt := fmt.Sprintf(`请仔细审查以下回答，并找出可能的问题或改进空间：

# 原始任务:
%s

# 当前回答:
%s

请分析这个回答的质量，指出不足之处，并提出具体的改进建议。
如果回答已经很好，请回答"无需改进"。`, task, result)

	messages := []map[string]any{
		{"role": "system", "content": a.SystemPrompt},
		{"role": "user", "content": prompt},
	}
	return a.getLLMResponse(messages, kwargs)
}

func (a *ReflectionAgent) refineResult(task string, lastAttempt string, feedback string, kwargs map[string]any) (string, error) {
	prompt := fmt.Sprintf(`请根据反馈意见改进你的回答：

# 原始任务:
%s

# 上一轮回答:
%s

# 反馈意见:
%s

请提供一个改进后的回答。`, task, lastAttempt, feedback)

	messages := []map[string]any{
		{"role": "system", "content": a.SystemPrompt},
		{"role": "user", "content": prompt},
	}
	return a.getLLMResponse(messages, kwargs)
}

func (a *ReflectionAgent) getLLMResponse(messages []map[string]any, kwargs map[string]any) (string, error) {
	if !a.EnableToolCalling || a.ToolRegistry == nil {
		llmResponse, err := a.LLM.Invoke(messages, kwargs)
		if err != nil {
			return "", err
		}
		return llmResponse.Content, nil
	}

	toolSchemas := a.BuildToolSchemas()
	currentIteration := 0

	for currentIteration < a.MaxToolIterations {
		currentIteration++
		response, err := a.LLM.InvokeWithTools(messages, toolSchemas, "auto", kwargs)
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
			result := a.ExecuteToolCall(toolCall.Name, arguments)

			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      result,
			})
		}
	}

	if currentIteration >= a.MaxToolIterations {
		llmResponse, err := a.LLM.Invoke(messages, kwargs)
		if err != nil {
			return "", err
		}
		return llmResponse.Content, nil
	}
	return "", nil
}

func (a *ReflectionAgent) ArunStream(inputText string, kwargs map[string]any, hooks ...core.Hooks) <-chan core.AgentEvent {
	activeHooks := core.Hooks{}
	if len(hooks) > 0 {
		activeHooks = hooks[0]
	}

	out := make(chan core.AgentEvent, 32)
	go func() {
		defer close(out)

		if activeHooks.OnStart != nil {
			_ = activeHooks.OnStart(core.NewAgentEvent(core.AgentStart, a.Name, map[string]any{
				"input_text": inputText,
			}))
		}
		out <- core.NewAgentEvent(core.AgentStart, a.Name, map[string]any{
			"input_text": inputText,
		})

		emitError := func(err error) {
			if activeHooks.OnError != nil {
				_ = activeHooks.OnError(core.NewAgentEvent(core.AgentError, a.Name, map[string]any{
					"error":      err.Error(),
					"error_type": "AgentError",
				}))
			}
			out <- core.NewAgentEvent(core.AgentError, a.Name, map[string]any{
				"error":      err.Error(),
				"error_type": "AgentError",
			})
		}

		out <- core.NewAgentEvent(core.StepStart, a.Name, map[string]any{
			"phase":       "initial_execution",
			"description": "生成初始回答",
		})

		initialMessages := []map[string]any{}
		if a.SystemPrompt != "" {
			initialMessages = append(initialMessages, map[string]any{"role": "system", "content": a.SystemPrompt})
		}
		for _, msg := range a.GetHistory() {
			initialMessages = append(initialMessages, map[string]any{"role": msg.Role, "content": msg.Content})
		}
		initialMessages = append(initialMessages, map[string]any{"role": "user", "content": inputText})

		initialResponse, err := streamLLMResponse(a.LLM, initialMessages, kwargs, func(chunk string) {
			out <- core.NewAgentEvent(core.LLMChunk, a.Name, map[string]any{
				"chunk": chunk,
				"phase": "execution",
			})
		})
		if err != nil {
			emitError(err)
			return
		}

		out <- core.NewAgentEvent(core.StepFinish, a.Name, map[string]any{
			"phase":  "initial_execution",
			"result": initialResponse,
		})

		currentResponse := initialResponse
		for i := 0; i < a.MaxIterations; i++ {
			iteration := i + 1
			out <- core.NewAgentEvent(core.StepStart, a.Name, map[string]any{
				"phase":       "reflection",
				"iteration":   iteration,
				"description": fmt.Sprintf("第 %d 次反思", iteration),
			})

			reflectionPrompt := a.buildReflectionPrompt(inputText, currentResponse)
			reflectionMessages := []map[string]any{
				{"role": "user", "content": reflectionPrompt},
			}
			reflection, err := streamLLMResponse(a.LLM, reflectionMessages, kwargs, func(chunk string) {
				out <- core.NewAgentEvent(core.Thinking, a.Name, map[string]any{
					"chunk":     chunk,
					"phase":     "reflection",
					"iteration": iteration,
				})
			})
			if err != nil {
				emitError(err)
				return
			}

			out <- core.NewAgentEvent(core.StepFinish, a.Name, map[string]any{
				"phase":      "reflection",
				"iteration":  iteration,
				"reflection": reflection,
			})

			out <- core.NewAgentEvent(core.StepStart, a.Name, map[string]any{
				"phase":       "refinement",
				"iteration":   iteration,
				"description": fmt.Sprintf("第 %d 次优化", iteration),
			})

			refinementPrompt := a.buildRefinementPrompt(inputText, currentResponse, reflection)
			refinementMessages := []map[string]any{
				{"role": "user", "content": refinementPrompt},
			}
			refinedResponse, err := streamLLMResponse(a.LLM, refinementMessages, kwargs, func(chunk string) {
				out <- core.NewAgentEvent(core.LLMChunk, a.Name, map[string]any{
					"chunk":     chunk,
					"phase":     "refinement",
					"iteration": iteration,
				})
			})
			if err != nil {
				emitError(err)
				return
			}

			out <- core.NewAgentEvent(core.StepFinish, a.Name, map[string]any{
				"phase":     "refinement",
				"iteration": iteration,
				"result":    refinedResponse,
			})

			currentResponse = refinedResponse
		}

		if activeHooks.OnFinish != nil {
			_ = activeHooks.OnFinish(core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
				"result":           currentResponse,
				"total_iterations": a.MaxIterations,
			}))
		}
		out <- core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
			"result":           currentResponse,
			"total_iterations": a.MaxIterations,
		})

		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(currentResponse, core.MessageRoleAssistant, nil))
	}()
	return out
}

func (a *ReflectionAgent) buildReflectionPrompt(task string, result string) string {
	return fmt.Sprintf(`请仔细审查以下回答，并找出可能的问题或改进空间：

# 原始任务:
%s

# 当前回答:
%s

请分析这个回答的质量，指出不足之处，并提出具体的改进建议。
如果回答已经很好，请回答"无需改进"。`, task, result)
}

func (a *ReflectionAgent) buildRefinementPrompt(task string, lastAttempt string, feedback string) string {
	return fmt.Sprintf(`请根据反馈意见改进你的回答：

# 原始任务:
%s

# 上一轮回答:
%s

# 反馈意见:
%s

请提供一个改进后的回答。`, task, lastAttempt, feedback)
}

func (a *ReflectionAgent) String() string {
	return fmt.Sprintf("ReflectionAgent(name=%s)", a.Name)
}
