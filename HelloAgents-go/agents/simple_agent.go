package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// SimpleAgent 是简单的对话 Agent，支持可选的工具调用。
// 使用自定义工具调用格式: [TOOL_CALL:tool_name:parameters]
// 嵌入 BaseAgent 以获得基础字段和方法，与 Python 继承 Agent 对应。
type SimpleAgent struct {
	*core.BaseAgent
	toolRegistry      *tools.ToolRegistry
	enableToolCalling bool
}

// NewSimpleAgent 创建 SimpleAgent。
func NewSimpleAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
) *SimpleAgent {
	return &SimpleAgent{
		BaseAgent:         core.NewBaseAgent(name, llm, systemPrompt, nil),
		toolRegistry:      toolRegistry,
		enableToolCalling: toolRegistry != nil,
	}
}

// Run 执行 Agent。
func (a *SimpleAgent) Run(ctx context.Context, inputText string) (string, error) {
	// 构建消息列表
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
	}

	// 添加历史消息
	for _, msg := range a.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}

	// 添加当前输入
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	// 如果启用工具调用，添加工具描述到系统提示
	if a.enableToolCalling && a.toolRegistry.Count() > 0 {
		toolDesc := a.toolRegistry.GetToolsDescription()
		messages[0] = core.ChatMessage{
			Role:    core.RoleSystem,
			Content: a.SystemPrompt + "\n\n" + toolDesc + "\n\nWhen you need to use a tool, use the format: [TOOL_CALL:tool_name:parameters]",
		}
	}

	// 调用 LLM
	response, err := a.LLM.Invoke(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 处理工具调用
	if a.enableToolCalling {
		response, err = a.processToolCalls(response)
		if err != nil {
			return "", fmt.Errorf("工具调用处理失败: %w", err)
		}
	}

	// 添加消息到历史记录
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))

	return response, nil
}

// processToolCalls 处理响应中的工具调用。
func (a *SimpleAgent) processToolCalls(response string) (string, error) {
	maxIterations := 10
	iterations := 0

	for iterations < maxIterations {
		iterations++

		// 检查是否有工具调用
		toolName, paramsStr, found := tools.ParseToolCall(response)
		if !found {
			break
		}

		// 解析参数
		var params map[string]interface{}
		var err error

		if strings.HasPrefix(paramsStr, "{") || strings.HasPrefix(paramsStr, "[") {
			params, err = tools.ConvertParameters(paramsStr)
		} else {
			params = map[string]interface{}{"input": paramsStr}
		}

		if err != nil {
			return "", fmt.Errorf("解析工具参数失败: %w", err)
		}

		// 执行工具
		result, err := a.toolRegistry.ExecuteTool(toolName, params)
		if err != nil {
			return "", fmt.Errorf("工具执行失败: %w", err)
		}

		// 构建后续消息
		messages := []core.ChatMessage{
			{Role: core.RoleSystem, Content: a.SystemPrompt},
		}
		for _, msg := range a.GetHistory() {
			messages = append(messages, msg.ToChatMessage())
		}
		messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: response})
		messages = append(messages, core.ChatMessage{
			Role:    core.RoleUser,
			Content: fmt.Sprintf("工具 %s 返回: %s\n\n请基于此结果继续。", toolName, result),
		})

		// 获取后续响应
		response, err = a.LLM.Invoke(context.Background(), messages, nil)
		if err != nil {
			return "", fmt.Errorf("后续 LLM 调用失败: %w", err)
		}
	}

	return response, nil
}

// StreamRun 流式执行 Agent。
func (a *SimpleAgent) StreamRun(ctx context.Context, inputText string) (<-chan string, <-chan error) {
	// 构建消息列表
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SystemPrompt},
	}
	for _, msg := range a.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	// 如果启用工具调用，添加工具描述
	if a.enableToolCalling && a.toolRegistry.Count() > 0 {
		toolDesc := a.toolRegistry.GetToolsDescription()
		messages[0] = core.ChatMessage{
			Role:    core.RoleSystem,
			Content: a.SystemPrompt + "\n\n" + toolDesc + "\n\nWhen you need to use a tool, use the format: [TOOL_CALL:tool_name:parameters]",
		}
	}

	// 流式调用 LLM
	streamCh, errCh := a.LLM.Think(ctx, messages, nil)

	// 创建输出通道
	outCh := make(chan string, 32)
	outErrCh := make(chan error, 1)

	go func() {
		defer close(outCh)
		defer close(outErrCh)

		var fullResponse strings.Builder
		for {
			select {
			case <-ctx.Done():
				outErrCh <- ctx.Err()
				return
			case chunk, ok := <-streamCh:
				if !ok {
					// 流结束，保存到历史
					a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
					a.AddMessage(core.NewMessage(fullResponse.String(), core.RoleAssistant, core.Time{}, nil))
					return
				}
				fullResponse.WriteString(chunk)
				outCh <- chunk
			case err, ok := <-errCh:
				if ok && err != nil {
					outErrCh <- err
					return
				}
			}
		}
	}()

	return outCh, outErrCh
}

// AddTool 注册工具。
func (a *SimpleAgent) AddTool(tool tools.Tool, autoExpand bool) error {
	if a.toolRegistry == nil {
		return fmt.Errorf("工具注册表未初始化")
	}
	return a.toolRegistry.RegisterTool(tool, autoExpand)
}

// RemoveTool 移除工具。
func (a *SimpleAgent) RemoveTool(name string) error {
	if a.toolRegistry == nil {
		return fmt.Errorf("工具注册表未初始化")
	}
	return a.toolRegistry.UnregisterTool(name)
}

// ListTools 列出所有工具。
func (a *SimpleAgent) ListTools() []string {
	if a.toolRegistry == nil {
		return []string{}
	}
	return a.toolRegistry.ListTools()
}

// HasTools 检查是否有可用工具。
func (a *SimpleAgent) HasTools() bool {
	return a.enableToolCalling && a.toolRegistry != nil
}

// EnableToolCalling 启用/禁用工具调用。
func (a *SimpleAgent) EnableToolCalling(enabled bool) {
	a.enableToolCalling = enabled
}

// IsToolCallingEnabled 返回工具调用是否启用。
func (a *SimpleAgent) IsToolCallingEnabled() bool {
	return a.enableToolCalling
}

// ExtractToolCalls 从响应中提取所有工具调用。
func (a *SimpleAgent) ExtractToolCalls(response string) []ToolCall {
	re := regexp.MustCompile(`\[TOOL_CALL:([^:]+):([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(response, -1)

	calls := make([]ToolCall, 0, len(matches))
	for _, match := range matches {
		if len(match) == 3 {
			calls = append(calls, ToolCall{
				Name:       match[1],
				Parameters: match[2],
			})
		}
	}
	return calls
}

// HasToolCall 检查响应是否包含工具调用。
func (a *SimpleAgent) HasToolCall(response string) bool {
	_, _, found := tools.ParseToolCall(response)
	return found
}

// StripToolCalls 移除响应中的工具调用标记。
func (a *SimpleAgent) StripToolCalls(response string) string {
	re := regexp.MustCompile(`\[TOOL_CALL:[^\]]+\]`)
	return re.ReplaceAllString(response, "")
}

// ToolCall 表示单个工具调用。
type ToolCall struct {
	Name       string
	Parameters string
}
