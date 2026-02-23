package agents

import (
	"fmt"
	"strings"
	"time"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

// SimpleAgent mirrors hello_agents.agents.simple_agent.SimpleAgent.
type SimpleAgent struct {
	*core.BaseAgent
	EnableToolCalling bool
	MaxToolIterations int
}

func NewSimpleAgent(name string, llm *core.HelloAgentsLLM, systemPrompt string, config *core.Config, toolRegistry *tools.ToolRegistry) (*SimpleAgent, error) {
	base, err := core.NewBaseAgent(name, llm, systemPrompt, config, toolRegistry)
	if err != nil {
		return nil, err
	}
	agent := &SimpleAgent{
		BaseAgent:         base,
		EnableToolCalling: toolRegistry != nil,
		MaxToolIterations: 3,
	}
	base.AgentType = "SimpleAgent"
	autoRegisterBuiltinTools(base)
	return agent, nil
}

func (a *SimpleAgent) Run(inputText string, kwargs map[string]any) (string, error) {
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	startTime := time.Now()

	maxToolIterations := a.MaxToolIterations
	if raw, ok := kwargs["max_tool_iterations"]; ok {
		maxToolIterations = intFromAny(raw)
	}
	if maxToolIterations <= 0 {
		maxToolIterations = 3
	}

	llmKwargs := cloneMapWithoutKeys(kwargs, "max_tool_iterations")
	messages := a.buildMessages(inputText)
	if a.TraceLogger != nil {
		a.TraceLogger.LogEvent("message_written", map[string]any{
			"role":    "user",
			"content": inputText,
		}, nil)
	}

	if !a.EnableToolCalling || a.ToolRegistry == nil {
		llmResponse, err := a.LLM.Invoke(messages, llmKwargs)
		if err != nil {
			return "", err
		}
		responseText := llmResponse.Content
		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(responseText, core.MessageRoleAssistant, nil))
		if a.TraceLogger != nil {
			a.TraceLogger.LogEvent("session_end", map[string]any{
				"duration":     time.Since(startTime).Seconds(),
				"final_answer": responseText,
				"status":       "success",
				"usage": map[string]any{
					"prompt_tokens":     llmResponse.Usage["prompt_tokens"],
					"completion_tokens": llmResponse.Usage["completion_tokens"],
					"total_tokens":      llmResponse.Usage["total_tokens"],
				},
				"latency_ms": llmResponse.LatencyMS,
			}, nil)
			_ = a.TraceLogger.Finalize()
		}
		return responseText, nil
	}

	toolSchemas := a.BuildToolSchemas()
	currentIteration := 0
	finalResponse := ""

	for currentIteration < maxToolIterations {
		currentIteration++
		response, err := a.LLM.InvokeWithTools(messages, toolSchemas, "auto", llmKwargs)
		if err != nil {
			if a.TraceLogger != nil {
				step := currentIteration
				a.TraceLogger.LogEvent("error", map[string]any{
					"error_type": "LLM_ERROR",
					"message":    err.Error(),
				}, &step)
			}
			break
		}

		content, toolCalls := extractToolCallsAndContent(response)
		if a.TraceLogger != nil {
			step := currentIteration
			a.TraceLogger.LogEvent("model_output", map[string]any{
				"content":    content,
				"tool_calls": len(toolCalls),
				"usage":      usageFromLLMRawResponse(response),
			}, &step)
		}
		if len(toolCalls) == 0 {
			if strings.TrimSpace(content) == "" {
				finalResponse = "抱歉，我无法回答这个问题。"
			} else {
				finalResponse = content
			}
			break
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
				step := currentIteration
				a.TraceLogger.LogEvent("tool_call", map[string]any{
					"tool_name":    toolCall.Name,
					"tool_call_id": toolCall.ID,
					"args":         arguments,
				}, &step)
			}
			result := a.ExecuteToolCall(toolCall.Name, arguments)
			if a.TraceLogger != nil {
				step := currentIteration
				a.TraceLogger.LogEvent("tool_result", map[string]any{
					"tool_name":    toolCall.Name,
					"tool_call_id": toolCall.ID,
					"result":       result,
				}, &step)
			}

			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      result,
			})
		}
	}

	if currentIteration >= maxToolIterations && finalResponse == "" {
		llmResponse, err := a.LLM.Invoke(messages, llmKwargs)
		if err != nil {
			return "", err
		}
		finalResponse = llmResponse.Content
	}

	a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
	a.AddMessage(core.NewMessage(finalResponse, core.MessageRoleAssistant, nil))
	if a.TraceLogger != nil {
		a.TraceLogger.LogEvent("session_end", map[string]any{
			"duration":     time.Since(startTime).Seconds(),
			"total_steps":  currentIteration,
			"final_answer": finalResponse,
			"status":       "success",
		}, nil)
		_ = a.TraceLogger.Finalize()
	}
	return finalResponse, nil
}

func (a *SimpleAgent) buildMessages(inputText string) []map[string]any {
	messages := make([]map[string]any, 0, len(a.GetHistory())+2)
	if strings.TrimSpace(a.SystemPrompt) != "" {
		messages = append(messages, map[string]any{"role": "system", "content": a.SystemPrompt})
	}
	for _, msg := range a.GetHistory() {
		messages = append(messages, map[string]any{"role": msg.Role, "content": msg.Content})
	}
	messages = append(messages, map[string]any{"role": "user", "content": inputText})
	return messages
}

func (a *SimpleAgent) AddTool(tool tools.Tool, autoExpand bool) {
	if a.ToolRegistry == nil {
		a.ToolRegistry = tools.NewToolRegistry(nil)
		a.EnableToolCalling = true
	}
	a.ToolRegistry.RegisterTool(tool, autoExpand)
}

func (a *SimpleAgent) RemoveTool(toolName string) bool {
	if a.ToolRegistry == nil {
		return false
	}
	return a.ToolRegistry.UnregisterTool(toolName)
}

func (a *SimpleAgent) ListTools() []string {
	if a.ToolRegistry == nil {
		return []string{}
	}
	return a.ToolRegistry.ListTools()
}

func (a *SimpleAgent) HasTools() bool {
	return a.EnableToolCalling && a.ToolRegistry != nil
}

func (a *SimpleAgent) GetToolRegistry() *tools.ToolRegistry {
	return a.ToolRegistry
}

func (a *SimpleAgent) StreamRun(inputText string, kwargs map[string]any) (<-chan string, <-chan error) {
	messages := a.buildMessages(inputText)
	chunkStream, streamErr := a.LLM.StreamInvoke(messages, kwargs)

	out := make(chan string)
	errOut := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errOut)

		full := ""
		for chunk := range chunkStream {
			full += chunk
			out <- chunk
		}

		for err := range streamErr {
			if err != nil {
				errOut <- err
				return
			}
		}

		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(full, core.MessageRoleAssistant, nil))
	}()

	return out, errOut
}

func (a *SimpleAgent) ArunStream(inputText string, kwargs map[string]any) <-chan core.AgentEvent {
	out := make(chan core.AgentEvent, 16)
	go func() {
		defer close(out)

		out <- core.NewAgentEvent(core.AgentStart, a.Name, map[string]any{
			"input_text": inputText,
		})

		messages := []map[string]any{}
		if strings.TrimSpace(a.SystemPrompt) != "" {
			messages = append(messages, map[string]any{"role": "system", "content": a.SystemPrompt})
		}
		for _, msg := range a.GetHistory() {
			messages = append(messages, map[string]any{"role": msg.Role, "content": msg.Content})
		}
		messages = append(messages, map[string]any{"role": "user", "content": inputText})

		fullResponse, err := streamLLMResponse(a.LLM, messages, kwargs, func(chunk string) {
			out <- core.NewAgentEvent(core.LLMChunk, a.Name, map[string]any{
				"chunk": chunk,
			})
		})
		if err != nil {
			out <- core.NewAgentEvent(core.AgentError, a.Name, map[string]any{
				"error":      err.Error(),
				"error_type": "AgentError",
			})
			return
		}

		out <- core.NewAgentEvent(core.AgentFinish, a.Name, map[string]any{
			"result": fullResponse,
		})

		a.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		a.AddMessage(core.NewMessage(fullResponse, core.MessageRoleAssistant, nil))
	}()
	return out
}

func (a *SimpleAgent) String() string {
	return fmt.Sprintf("SimpleAgent(name=%s)", a.Name)
}

func (a *SimpleAgent) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	return runAsSubagent(a, task, toolFilter, returnSummary, maxStepsOverride)
}

func cloneMapWithoutKeys(source map[string]any, keys ...string) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	skip := map[string]struct{}{}
	for _, key := range keys {
		skip[key] = struct{}{}
	}
	out := map[string]any{}
	for key, value := range source {
		if _, ok := skip[key]; ok {
			continue
		}
		out[key] = value
	}
	return out
}
