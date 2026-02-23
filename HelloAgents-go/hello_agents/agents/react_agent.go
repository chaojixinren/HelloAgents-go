package agents

import (
	"fmt"
	"time"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

const defaultReActSystemPrompt = `你是一个具备推理和行动能力的 AI 助手。

## 工作流程
你可以通过调用工具来完成任务：

1. **Thought 工具**：用于记录你的推理过程和分析
   - 在需要思考时调用
   - 参数：reasoning（你的推理内容）

2. **业务工具**：用于获取信息或执行操作
   - 根据任务需求选择合适的工具
   - 可以多次调用不同工具

3. **Finish 工具**：用于返回最终答案
   - 当你有足够信息得出结论时调用
   - 参数：answer（最终答案）

## 重要提醒
- 主动使用 Thought 工具记录推理过程
- 可以多次调用工具获取信息
- 只有在确信有足够信息时才调用 Finish
`

type builtinToolResult struct {
	Content     string
	Finished    bool
	FinalAnswer string
}

// ReActAgent mirrors hello_agents.agents.react_agent.ReActAgent.
type ReActAgent struct {
	*core.BaseAgent
	MaxSteps     int
	builtinTools map[string]struct{}
}

func NewReActAgent(name string, llm *core.HelloAgentsLLM, systemPrompt string, config *core.Config, toolRegistry *tools.ToolRegistry, maxSteps int) (*ReActAgent, error) {
	if toolRegistry == nil {
		toolRegistry = tools.NewToolRegistry(nil)
	}
	if systemPrompt == "" {
		systemPrompt = defaultReActSystemPrompt
	}
	if maxSteps <= 0 {
		maxSteps = 5
	}

	base, err := core.NewBaseAgent(name, llm, systemPrompt, config, toolRegistry)
	if err != nil {
		return nil, err
	}
	agent := &ReActAgent{
		BaseAgent: base,
		MaxSteps:  maxSteps,
		builtinTools: map[string]struct{}{
			"Thought": {},
			"Finish":  {},
		},
	}
	base.AgentType = "ReActAgent"
	base.MaxSteps = maxSteps
	autoRegisterBuiltinTools(base)
	return agent, nil
}

func (a *ReActAgent) AddTool(tool tools.Tool) {
	if a.ToolRegistry == nil {
		a.ToolRegistry = tools.NewToolRegistry(nil)
	}
	a.ToolRegistry.RegisterTool(tool, true)
}

func (a *ReActAgent) Run(inputText string, kwargs map[string]any) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	startTime := time.Now()

	messages := a.buildMessages(inputText)
	toolSchemas := a.buildToolSchemas()
	llmKwargs := cloneMapWithoutKeys(kwargs)
	if a.TraceLogger != nil {
		a.TraceLogger.LogEvent("message_written", map[string]any{
			"role":    "user",
			"content": inputText,
		}, nil)
	}

	currentStep := 0
	for currentStep < a.MaxSteps {
		currentStep++

		response, err := a.LLM.InvokeWithTools(messages, toolSchemas, "auto", llmKwargs)
		if err != nil {
			if a.TraceLogger != nil {
				step := currentStep
				a.TraceLogger.LogEvent("error", map[string]any{
					"error_type": "LLM_ERROR",
					"message":    err.Error(),
				}, &step)
			}
			break
		}

		content, toolCalls := extractToolCallsAndContent(response)
		if a.TraceLogger != nil {
			step := currentStep
			a.TraceLogger.LogEvent("model_output", map[string]any{
				"content":    content,
				"tool_calls": len(toolCalls),
				"usage":      usageFromLLMRawResponse(response),
			}, &step)
		}
		if len(toolCalls) == 0 {
			finalAnswer := content
			if finalAnswer == "" {
				finalAnswer = "抱歉，我无法回答这个问题。"
			}
			a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
			a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
			if a.TraceLogger != nil {
				a.TraceLogger.LogEvent("session_end", map[string]any{
					"duration":     time.Since(startTime).Seconds(),
					"total_steps":  currentStep,
					"final_answer": finalAnswer,
					"status":       "success",
				}, nil)
				_ = a.TraceLogger.Finalize()
			}
			return finalAnswer, nil
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
			if a.TraceLogger != nil {
				step := currentStep
				a.TraceLogger.LogEvent("tool_call", map[string]any{
					"tool_name":    toolCall.Name,
					"tool_call_id": toolCall.ID,
					"args":         arguments,
				}, &step)
			}

			if _, isBuiltin := a.builtinTools[toolCall.Name]; isBuiltin {
				result := a.handleBuiltinTool(toolCall.Name, arguments)
				if a.TraceLogger != nil {
					step := currentStep
					a.TraceLogger.LogEvent("tool_result", map[string]any{
						"tool_name":    toolCall.Name,
						"tool_call_id": toolCall.ID,
						"status":       "success",
						"result":       result.Content,
					}, &step)
				}
				if toolCall.Name == "Finish" && result.Finished {
					a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
					a.AddMessage(core.NewMessage(result.FinalAnswer, core.MessageRoleAssistant, nil))
					if a.TraceLogger != nil {
						a.TraceLogger.LogEvent("session_end", map[string]any{
							"duration":     time.Since(startTime).Seconds(),
							"total_steps":  currentStep,
							"final_answer": result.FinalAnswer,
							"status":       "success",
						}, nil)
						_ = a.TraceLogger.Finalize()
					}
					return result.FinalAnswer, nil
				}
				messages = append(messages, map[string]any{
					"role":         "tool",
					"tool_call_id": toolCall.ID,
					"content":      result.Content,
				})
				continue
			}

			toolResult := a.ExecuteToolCall(toolCall.Name, arguments)
			if a.TraceLogger != nil {
				step := currentStep
				a.TraceLogger.LogEvent("tool_result", map[string]any{
					"tool_name":    toolCall.Name,
					"tool_call_id": toolCall.ID,
					"result":       toolResult,
				}, &step)
			}
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      toolResult,
			})
		}
	}

	finalAnswer := "抱歉，我无法在限定步数内完成这个任务。"
	a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
	if a.TraceLogger != nil {
		a.TraceLogger.LogEvent("session_end", map[string]any{
			"duration":     time.Since(startTime).Seconds(),
			"total_steps":  currentStep,
			"final_answer": finalAnswer,
			"status":       "timeout",
		}, nil)
		_ = a.TraceLogger.Finalize()
	}
	return finalAnswer, nil
}

func (a *ReActAgent) buildMessages(inputText string) []map[string]any {
	messages := make([]map[string]any, 0, len(a.GetHistory())+2)
	if a.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": a.SystemPrompt})
	}
	for _, msg := range a.GetHistory() {
		messages = append(messages, map[string]any{"role": msg.Role, "content": msg.Content})
	}
	messages = append(messages, map[string]any{"role": "user", "content": inputText})
	return messages
}

func (a *ReActAgent) buildToolSchemas() []map[string]any {
	schemas := []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "Thought",
				"description": "分析问题，制定策略，记录推理过程。在需要思考时调用此工具。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"reasoning": map[string]any{
							"type":        "string",
							"description": "你的推理过程和分析",
						},
					},
					"required": []string{"reasoning"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "Finish",
				"description": "当你有足够信息得出结论时，使用此工具返回最终答案。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"answer": map[string]any{
							"type":        "string",
							"description": "最终答案",
						},
					},
					"required": []string{"answer"},
				},
			},
		},
	}

	if a.ToolRegistry != nil {
		schemas = append(schemas, a.BuildToolSchemas()...)
	}
	return schemas
}

func (a *ReActAgent) handleBuiltinTool(toolName string, arguments map[string]any) builtinToolResult {
	switch toolName {
	case "Thought":
		reasoning, _ := arguments["reasoning"].(string)
		return builtinToolResult{Content: "推理: " + reasoning, Finished: false}
	case "Finish":
		answer, _ := arguments["answer"].(string)
		return builtinToolResult{Content: "最终答案: " + answer, Finished: true, FinalAnswer: answer}
	default:
		return builtinToolResult{Content: "未知的内置工具: " + toolName, Finished: false}
	}
}

func (a *ReActAgent) GetToolRegistry() *tools.ToolRegistry {
	return a.ToolRegistry
}

func (a *ReActAgent) GetMaxSteps() int {
	return a.MaxSteps
}

func (a *ReActAgent) SetMaxSteps(v int) {
	if v <= 0 {
		return
	}
	a.MaxSteps = v
	a.BaseAgent.MaxSteps = v
}

func (a *ReActAgent) String() string {
	return fmt.Sprintf("ReActAgent(name=%s)", a.Name)
}

func (a *ReActAgent) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	return runAsSubagent(a, task, toolFilter, returnSummary, maxStepsOverride)
}
