package agents

import (
	"fmt"
	"sync"
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

type reactToolExecutionResult struct {
	ToolName   string
	ToolCallID string
	Result     map[string]any
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
	base.SetRunDelegate(agent.Run)
	base.SetMaxStepAccessors(
		func() int {
			return agent.MaxSteps
		},
		func(v int) {
			if v <= 0 {
				return
			}
			agent.MaxSteps = v
			base.MaxSteps = v
		},
	)
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
	totalTokens := 0
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

		usage := usageFromLLMRawResponse(response)
		totalTokens += intFromAny(usage["total_tokens"])

		content, toolCalls := extractToolCallsAndContent(response)
		if a.TraceLogger != nil {
			step := currentStep
			a.TraceLogger.LogEvent("model_output", map[string]any{
				"content":    content,
				"tool_calls": len(toolCalls),
				"usage":      usage,
			}, &step)
		}
		if len(toolCalls) == 0 {
			finalAnswer := content
			if finalAnswer == "" {
				finalAnswer = "抱歉，我无法回答这个问题。"
			}
			a.SessionMetadata["total_steps"] = currentStep
			a.SessionMetadata["total_tokens"] = totalTokens
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
					a.SessionMetadata["total_steps"] = currentStep
					a.SessionMetadata["total_tokens"] = totalTokens
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
	a.SessionMetadata["total_steps"] = currentStep
	a.SessionMetadata["total_tokens"] = totalTokens
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

func (a *ReActAgent) Arun(inputText string, hooks core.Hooks, kwargs map[string]any) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	startTime := time.Now()
	a.emitHook(core.AgentStart, hooks.OnStart, map[string]any{"input_text": inputText})

	messages := a.buildMessages(inputText)
	toolSchemas := a.buildToolSchemas()
	currentStep := 0
	totalTokens := 0

	if a.TraceLogger != nil {
		a.TraceLogger.LogEvent("message_written", map[string]any{
			"role":    "user",
			"content": inputText,
		}, nil)
	}

	for currentStep < a.MaxSteps {
		currentStep++
		a.emitHook(core.StepStart, hooks.OnStep, map[string]any{"step": currentStep})

		response, err := a.LLM.AInvokeWithTools(messages, toolSchemas, "auto", kwargs)
		if err != nil {
			a.emitHook(core.AgentError, hooks.OnError, map[string]any{
				"error": err.Error(),
				"step":  currentStep,
			})
			break
		}

		usage := usageFromLLMRawResponse(response)
		totalTokens += intFromAny(usage["total_tokens"])

		content, toolCalls := extractToolCallsAndContent(response)
		if a.TraceLogger != nil {
			step := currentStep
			a.TraceLogger.LogEvent("model_output", map[string]any{
				"content":    content,
				"tool_calls": len(toolCalls),
				"usage":      usage,
			}, &step)
		}

		if len(toolCalls) == 0 {
			finalAnswer := content
			if finalAnswer == "" {
				finalAnswer = "抱歉，我无法回答这个问题。"
			}

			a.SessionMetadata["total_steps"] = currentStep
			a.SessionMetadata["total_tokens"] = totalTokens
			a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
			a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))

			a.emitHook(core.AgentFinish, hooks.OnFinish, map[string]any{
				"result":       finalAnswer,
				"total_steps":  currentStep,
				"total_tokens": totalTokens,
			})

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

		toolResults := a.executeToolsAsync(toolCalls, currentStep, hooks.OnToolCall)
		for _, item := range toolResults {
			if item.ToolName == "Finish" && toBool(item.Result["finished"]) {
				finalAnswer := fmt.Sprintf("%v", item.Result["final_answer"])
				if finalAnswer == "" || finalAnswer == "<nil>" {
					finalAnswer = fmt.Sprintf("%v", item.Result["content"])
				}
				a.SessionMetadata["total_steps"] = currentStep
				a.SessionMetadata["total_tokens"] = totalTokens
				a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
				a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))

				a.emitHook(core.AgentFinish, hooks.OnFinish, map[string]any{
					"result":       finalAnswer,
					"total_steps":  currentStep,
					"total_tokens": totalTokens,
				})

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
				"role":         "tool",
				"tool_call_id": item.ToolCallID,
				"content":      fmt.Sprintf("%v", item.Result["content"]),
			})
		}

		a.emitHook(core.StepFinish, hooks.OnStep, map[string]any{
			"step":       currentStep,
			"tool_calls": len(toolCalls),
		})
	}

	finalAnswer := "抱歉，我无法在限定步数内完成这个任务。"
	a.SessionMetadata["total_steps"] = currentStep
	a.SessionMetadata["total_tokens"] = totalTokens
	a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
	a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))

	a.emitHook(core.AgentFinish, hooks.OnFinish, map[string]any{
		"result":       finalAnswer,
		"total_steps":  currentStep,
		"total_tokens": totalTokens,
		"status":       "timeout",
	})

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

func (a *ReActAgent) ArunStream(inputText string, kwargs map[string]any, hooks ...core.Hooks) <-chan core.AgentEvent {
	activeHooks := core.Hooks{}
	if len(hooks) > 0 {
		activeHooks = hooks[0]
	}

	out := make(chan core.AgentEvent, 64)
	go func() {
		defer close(out)

		a.emitHook(core.AgentStart, activeHooks.OnStart, map[string]any{
			"input_text": inputText,
		})
		out <- core.NewAgentEvent(core.AgentStart, a.Name, map[string]any{
			"input_text": inputText,
		})

		messages := a.buildMessages(inputText)
		toolSchemas := a.buildToolSchemas()
		currentStep := 0
		finalAnswer := ""

		for currentStep < a.MaxSteps {
			currentStep++
			a.emitHook(core.StepStart, activeHooks.OnStep, map[string]any{
				"step":      currentStep,
				"max_steps": a.MaxSteps,
			})
			out <- core.NewAgentEvent(core.StepStart, a.Name, map[string]any{
				"step":      currentStep,
				"max_steps": a.MaxSteps,
			})

			fullResponse, err := streamLLMResponse(a.LLM, messages, kwargs, func(chunk string) {
				out <- core.NewAgentEvent(core.LLMChunk, a.Name, map[string]any{
					"chunk": chunk,
					"step":  currentStep,
				})
			})
			if err != nil {
				a.emitHook(core.AgentError, activeHooks.OnError, map[string]any{
					"error": err.Error(),
					"step":  currentStep,
				})
				out <- core.NewAgentEvent(core.AgentError, a.Name, map[string]any{
					"error": err.Error(),
					"step":  currentStep,
				})
				break
			}

			response, err := a.LLM.InvokeWithTools(messages, toolSchemas, "auto", kwargs)
			if err != nil {
				a.emitHook(core.AgentError, activeHooks.OnError, map[string]any{
					"error": err.Error(),
					"step":  currentStep,
				})
				out <- core.NewAgentEvent(core.AgentError, a.Name, map[string]any{
					"error": err.Error(),
					"step":  currentStep,
				})
				break
			}

			content, toolCalls := extractToolCallsAndContent(response)
			if len(toolCalls) == 0 {
				finalAnswer = content
				if finalAnswer == "" {
					finalAnswer = fullResponse
				}
				if finalAnswer == "" {
					finalAnswer = "抱歉，我无法回答这个问题。"
				}
				a.emitHook(core.AgentFinish, activeHooks.OnFinish, map[string]any{
					"result":      finalAnswer,
					"total_steps": currentStep,
				})
				out <- core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
					"result":      finalAnswer,
					"total_steps": currentStep,
				})
				a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
				a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
				return
			}

			messages = append(messages, map[string]any{
				"role":       "assistant",
				"content":    content,
				"tool_calls": toOpenAIToolCallsPayload(toolCalls),
			})

			toolResults := a.executeToolsAsyncStream(toolCalls, currentStep, activeHooks.OnToolCall)
			for _, item := range toolResults {
				out <- core.NewAgentEvent(core.ToolResult, a.Name, map[string]any{
					"tool_name":    item.ToolName,
					"tool_call_id": item.ToolCallID,
					"result":       item.Result["content"],
					"step":         currentStep,
				})

				messages = append(messages, map[string]any{
					"role":         "tool",
					"tool_call_id": item.ToolCallID,
					"content":      fmt.Sprintf("%v", item.Result["content"]),
				})

				if item.ToolName == "Finish" {
					finalAnswer = fmt.Sprintf("%v", item.Result["content"])
					a.emitHook(core.AgentFinish, activeHooks.OnFinish, map[string]any{
						"result":      finalAnswer,
						"total_steps": currentStep,
					})
					out <- core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
						"result":      finalAnswer,
						"total_steps": currentStep,
					})
					a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
					a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
					return
				}
			}

			out <- core.NewAgentEvent(core.StepFinish, a.Name, map[string]any{
				"step": currentStep,
			})
			a.emitHook(core.StepFinish, activeHooks.OnStep, map[string]any{
				"step": currentStep,
			})
		}

		if finalAnswer == "" {
			finalAnswer = "抱歉，已达到最大步数限制，无法完成任务。"
		}
		a.emitHook(core.AgentFinish, activeHooks.OnFinish, map[string]any{
			"result":            finalAnswer,
			"total_steps":       currentStep,
			"max_steps_reached": true,
		})
		out <- core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
			"result":            finalAnswer,
			"total_steps":       currentStep,
			"max_steps_reached": true,
		})
		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(finalAnswer, core.MessageRoleAssistant, nil))
	}()
	return out
}

func (a *ReActAgent) executeToolsAsync(
	toolCalls []toolCallEnvelope,
	currentStep int,
	onToolCall core.LifecycleHook,
) []reactToolExecutionResult {
	results := make([]reactToolExecutionResult, 0, len(toolCalls))
	builtinCalls := make([]toolCallEnvelope, 0)
	userCalls := make([]toolCallEnvelope, 0)

	for _, tc := range toolCalls {
		if _, ok := a.builtinTools[tc.Name]; ok {
			builtinCalls = append(builtinCalls, tc)
		} else {
			userCalls = append(userCalls, tc)
		}
	}

	for _, tc := range builtinCalls {
		args := tc.Arguments
		if args == nil {
			args = map[string]any{}
		}
		if tc.ParseError != "" {
			results = append(results, reactToolExecutionResult{
				ToolName:   tc.Name,
				ToolCallID: tc.ID,
				Result: map[string]any{
					"content": "错误：参数格式不正确 - " + tc.ParseError,
				},
			})
			continue
		}

		a.emitHook(core.ToolCall, onToolCall, map[string]any{
			"tool_name":    tc.Name,
			"tool_call_id": tc.ID,
			"args":         args,
			"step":         currentStep,
		})

		builtin := a.handleBuiltinTool(tc.Name, args)
		if a.TraceLogger != nil {
			step := currentStep
			a.TraceLogger.LogEvent("tool_result", map[string]any{
				"tool_name":    tc.Name,
				"tool_call_id": tc.ID,
				"status":       "success",
				"result":       builtin.Content,
			}, &step)
		}

		results = append(results, reactToolExecutionResult{
			ToolName:   tc.Name,
			ToolCallID: tc.ID,
			Result: map[string]any{
				"content":      builtin.Content,
				"finished":     builtin.Finished,
				"final_answer": builtin.FinalAnswer,
			},
		})
	}

	if len(userCalls) > 0 {
		maxConcurrent := a.Config.MaxConcurrentTools
		if maxConcurrent <= 0 {
			maxConcurrent = 3
		}
		sem := make(chan struct{}, maxConcurrent)
		ordered := make([]reactToolExecutionResult, len(userCalls))
		var wg sync.WaitGroup

		for i, tc := range userCalls {
			wg.Add(1)
			go func(idx int, call toolCallEnvelope) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				args := call.Arguments
				if args == nil {
					args = map[string]any{}
				}
				if call.ParseError != "" {
					ordered[idx] = reactToolExecutionResult{
						ToolName:   call.Name,
						ToolCallID: call.ID,
						Result: map[string]any{
							"content": "错误：参数格式不正确 - " + call.ParseError,
						},
					}
					return
				}

				a.emitHook(core.ToolCall, onToolCall, map[string]any{
					"tool_name":    call.Name,
					"tool_call_id": call.ID,
					"args":         args,
					"step":         currentStep,
				})

				resultContent := fmt.Sprintf("❌ 工具 %s 不存在", call.Name)
				if tool := a.ToolRegistry.GetTool(call.Name); tool != nil {
					toolResponse := tool.ARunWithTiming(args)
					resultContent = toolResponse.Text
					if a.Truncator != nil {
						preview, _ := a.Truncator.Truncate(resultContent, call.Name)
						resultContent = preview
					}
				}

				if a.TraceLogger != nil {
					step := currentStep
					a.TraceLogger.LogEvent("tool_result", map[string]any{
						"tool_name":    call.Name,
						"tool_call_id": call.ID,
						"result":       resultContent,
					}, &step)
				}

				ordered[idx] = reactToolExecutionResult{
					ToolName:   call.Name,
					ToolCallID: call.ID,
					Result: map[string]any{
						"content": resultContent,
					},
				}
			}(i, tc)
		}
		wg.Wait()
		results = append(results, ordered...)
	}

	return results
}

func (a *ReActAgent) executeToolsAsyncStream(
	toolCalls []toolCallEnvelope,
	currentStep int,
	onToolCall core.LifecycleHook,
) []reactToolExecutionResult {
	results := make([]reactToolExecutionResult, 0, len(toolCalls))
	builtinCalls := make([]toolCallEnvelope, 0)
	userCalls := make([]toolCallEnvelope, 0)

	for _, tc := range toolCalls {
		if _, ok := a.builtinTools[tc.Name]; ok {
			builtinCalls = append(builtinCalls, tc)
		} else {
			userCalls = append(userCalls, tc)
		}
	}

	// Stream version keeps parity with python: builtin tools use stream-specific content.
	for _, tc := range builtinCalls {
		args := tc.Arguments
		if args == nil {
			args = map[string]any{}
		}
		if tc.ParseError != "" {
			results = append(results, reactToolExecutionResult{
				ToolName:   tc.Name,
				ToolCallID: tc.ID,
				Result: map[string]any{
					"content": "错误：参数格式不正确 - " + tc.ParseError,
				},
			})
			continue
		}

		a.emitHook(core.ToolCall, onToolCall, map[string]any{
			"tool_name":    tc.Name,
			"tool_call_id": tc.ID,
			"args":         args,
			"step":         currentStep,
		})

		resultContent := ""
		switch tc.Name {
		case "Thought":
			reasoning, _ := args["reasoning"].(string)
			resultContent = "已记录推理过程: " + reasoning
		case "Finish":
			answer, _ := args["answer"].(string)
			resultContent = answer
		default:
			resultContent = "未知的内置工具: " + tc.Name
		}

		results = append(results, reactToolExecutionResult{
			ToolName:   tc.Name,
			ToolCallID: tc.ID,
			Result: map[string]any{
				"content": resultContent,
			},
		})
	}

	if len(userCalls) == 0 {
		return results
	}

	maxConcurrent := a.Config.MaxConcurrentTools
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	sem := make(chan struct{}, maxConcurrent)
	ordered := make([]reactToolExecutionResult, len(userCalls))
	var wg sync.WaitGroup

	for i, tc := range userCalls {
		wg.Add(1)
		go func(idx int, call toolCallEnvelope) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			args := call.Arguments
			if args == nil {
				args = map[string]any{}
			}
			if call.ParseError != "" {
				ordered[idx] = reactToolExecutionResult{
					ToolName:   call.Name,
					ToolCallID: call.ID,
					Result: map[string]any{
						"content": "错误：参数格式不正确 - " + call.ParseError,
					},
				}
				return
			}

			a.emitHook(core.ToolCall, onToolCall, map[string]any{
				"tool_name":    call.Name,
				"tool_call_id": call.ID,
				"args":         args,
				"step":         currentStep,
			})

			resultContent := fmt.Sprintf("❌ 工具 %s 不存在", call.Name)
			if a.ToolRegistry != nil {
				if tool := a.ToolRegistry.GetTool(call.Name); tool != nil {
					toolResponse := tool.ARunWithTiming(args)
					resultContent = toolResponse.Text
					if a.Truncator != nil {
						preview, _ := a.Truncator.Truncate(resultContent, call.Name)
						resultContent = preview
					}
				}
			}

			ordered[idx] = reactToolExecutionResult{
				ToolName:   call.Name,
				ToolCallID: call.ID,
				Result: map[string]any{
					"content": resultContent,
				},
			}
		}(i, tc)
	}
	wg.Wait()
	results = append(results, ordered...)
	return results
}

func (a *ReActAgent) emitHook(eventType core.EventType, hook core.LifecycleHook, data map[string]any) {
	if hook == nil {
		return
	}
	_ = hook(core.NewAgentEvent(eventType, a.Name, data))
}

func toBool(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return value == "true" || value == "1" || value == "yes"
	case int:
		return value != 0
	case float64:
		return value != 0
	default:
		return false
	}
}

func (a *ReActAgent) buildMessages(inputText string) []map[string]any {
	messages := make([]map[string]any, 0, 2)
	if a.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": a.SystemPrompt})
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

func (a *ReActAgent) String() string {
	return fmt.Sprintf("ReActAgent(name=%s)", a.Name)
}
