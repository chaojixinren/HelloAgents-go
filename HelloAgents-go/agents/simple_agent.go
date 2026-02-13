package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// SimpleAgent 是简单的对话 Agent，支持可选的工具调用。
// 使用自定义工具调用格式: [TOOL_CALL:tool_name:parameters]
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
	enableToolCalling bool,
) *SimpleAgent {
	return &SimpleAgent{
		BaseAgent:         core.NewBaseAgent(name, llm, systemPrompt, nil),
		toolRegistry:      toolRegistry,
		enableToolCalling: enableToolCalling && toolRegistry != nil,
	}
}

// getEnhancedSystemPrompt 构建增强的系统提示词，包含工具信息。
func (a *SimpleAgent) getEnhancedSystemPrompt() string {
	basePrompt := a.SystemPrompt
	if basePrompt == "" {
		basePrompt = "你是一个有用的AI助手。"
	}

	if !a.enableToolCalling || a.toolRegistry == nil {
		return basePrompt
	}

	toolsDesc := a.toolRegistry.GetToolsDescription()
	if toolsDesc == "" || toolsDesc == "暂无可用工具" {
		return basePrompt
	}

	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n## 可用工具\n")
	sb.WriteString("你可以使用以下工具来帮助回答问题：\n")
	sb.WriteString(toolsDesc)
	sb.WriteString("\n\n## 工具调用格式\n")
	sb.WriteString("当需要使用工具时，请使用以下格式：\n")
	sb.WriteString("`[TOOL_CALL:{tool_name}:{parameters}]`\n\n")
	sb.WriteString("### 参数格式说明\n")
	sb.WriteString("1. **多个参数**：使用 `key=value` 格式，用逗号分隔\n")
	sb.WriteString("   示例：`[TOOL_CALL:calculator_multiply:a=12,b=8]`\n\n")
	sb.WriteString("2. **单个参数**：直接使用 `key=value`\n")
	sb.WriteString("   示例：`[TOOL_CALL:search:query=Python编程]`\n\n")
	sb.WriteString("3. **简单查询**：可以直接传入文本\n")
	sb.WriteString("   示例：`[TOOL_CALL:search:Python编程]`\n")

	return sb.String()
}

// parseToolCalls 解析文本中的工具调用。
func (a *SimpleAgent) parseToolCalls(text string) []map[string]string {
	pattern := regexp.MustCompile(`\[TOOL_CALL:([^:]+):([^\]]+)\]`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	calls := make([]map[string]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 3 {
			calls = append(calls, map[string]string{
				"tool_name":  strings.TrimSpace(match[1]),
				"parameters": strings.TrimSpace(match[2]),
				"original":   match[0],
			})
		}
	}
	return calls
}

// executeToolCall 执行工具调用。
func (a *SimpleAgent) executeToolCall(toolName, parameters string) string {
	if a.toolRegistry == nil {
		return "❌ 错误：未配置工具注册表"
	}

	tool := a.toolRegistry.GetTool(toolName)
	if tool == nil {
		return fmt.Sprintf("❌ 错误：未找到工具 '%s'", toolName)
	}

	paramDict := a.parseToolParameters(toolName, parameters)

	result, err := tool.Run(paramDict)
	if err != nil {
		return fmt.Sprintf("❌ 工具调用失败：%s", err)
	}
	return fmt.Sprintf("🔧 工具 %s 执行结果：\n%s", toolName, result)
}

// parseToolParameters 智能解析工具参数。
func (a *SimpleAgent) parseToolParameters(toolName, parameters string) map[string]interface{} {
	paramDict := make(map[string]interface{})
	parameters = strings.TrimSpace(parameters)

	// 尝试解析 JSON 格式
	if strings.HasPrefix(parameters, "{") {
		if err := json.Unmarshal([]byte(parameters), &paramDict); err == nil {
			return a.convertParameterTypes(toolName, paramDict)
		}
	}

	// key=value 格式
	if strings.Contains(parameters, "=") {
		if strings.Contains(parameters, ",") {
			pairs := strings.Split(parameters, ",")
			for _, pair := range pairs {
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					paramDict[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		} else {
			if kv := strings.SplitN(parameters, "=", 2); len(kv) == 2 {
				paramDict[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}

		paramDict = a.convertParameterTypes(toolName, paramDict)

		if _, ok := paramDict["action"]; !ok {
			paramDict = a.inferAction(toolName, paramDict)
		}
	} else {
		paramDict = a.inferSimpleParameters(toolName, parameters)
	}

	return paramDict
}

// convertParameterTypes 根据工具的参数定义转换参数类型。
func (a *SimpleAgent) convertParameterTypes(toolName string, paramDict map[string]interface{}) map[string]interface{} {
	if a.toolRegistry == nil {
		return paramDict
	}

	tool := a.toolRegistry.GetTool(toolName)
	if tool == nil {
		return paramDict
	}

	toolParams := tool.GetParameters()
	paramTypes := make(map[string]string)
	for _, p := range toolParams {
		paramTypes[p.Name] = p.Type
	}

	converted := make(map[string]interface{})
	for key, value := range paramDict {
		if paramType, ok := paramTypes[key]; ok {
			converted[key] = convertValue(value, paramType)
		} else {
			converted[key] = value
		}
	}

	return converted
}

// convertValue 转换单个值的类型。
func convertValue(value interface{}, paramType string) interface{} {
	strVal, ok := value.(string)
	if !ok {
		return value
	}

	switch paramType {
	case "number":
		if f, err := strconv.ParseFloat(strVal, 64); err == nil {
			return f
		}
	case "integer":
		if i, err := strconv.Atoi(strVal); err == nil {
			return i
		}
	case "boolean":
		lower := strings.ToLower(strVal)
		return lower == "true" || lower == "1" || lower == "yes"
	}

	return value
}

// inferAction 根据工具类型和参数推断 action。
func (a *SimpleAgent) inferAction(toolName string, paramDict map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range paramDict {
		result[k] = v
	}

	switch toolName {
	case "memory":
		if _, ok := result["recall"]; ok {
			result["action"] = "search"
			result["query"] = result["recall"]
			delete(result, "recall")
		} else if _, ok := result["store"]; ok {
			result["action"] = "add"
			result["content"] = result["store"]
			delete(result, "store")
		} else if _, ok := result["query"]; ok {
			result["action"] = "search"
		} else if _, ok := result["content"]; ok {
			result["action"] = "add"
		}
	case "rag":
		if _, ok := result["search"]; ok {
			result["action"] = "search"
			result["query"] = result["search"]
			delete(result, "search")
		} else if _, ok := result["query"]; ok {
			result["action"] = "search"
		} else if _, ok := result["text"]; ok {
			result["action"] = "add_text"
		}
	}

	return result
}

// inferSimpleParameters 为简单参数推断完整的参数字典。
func (a *SimpleAgent) inferSimpleParameters(toolName, parameters string) map[string]interface{} {
	switch toolName {
	case "rag":
		return map[string]interface{}{"action": "search", "query": parameters}
	case "memory":
		return map[string]interface{}{"action": "search", "query": parameters}
	default:
		return map[string]interface{}{"input": parameters}
	}
}

// Run 运行 Agent，支持多轮工具调用。
func (a *SimpleAgent) Run(ctx context.Context, inputText string, maxToolIterations int) (string, error) {
	if maxToolIterations <= 0 {
		maxToolIterations = 3
	}

	// 构建消息列表
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.getEnhancedSystemPrompt()},
	}

	// 添加历史消息
	for _, msg := range a.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}

	// 添加当前用户消息
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	// 如果没有启用工具调用，直接调用 LLM
	if !a.enableToolCalling {
		response, err := a.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}
		a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
		a.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))
		return response, nil
	}

	// 迭代处理，支持多轮工具调用
	var finalResponse string
	for i := 0; i < maxToolIterations; i++ {
		response, err := a.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		toolCalls := a.parseToolCalls(response)
		if len(toolCalls) == 0 {
			finalResponse = response
			break
		}

		// 执行所有工具调用
		var toolResults []string
		cleanResponse := response
		for _, call := range toolCalls {
			result := a.executeToolCall(call["tool_name"], call["parameters"])
			toolResults = append(toolResults, result)
			cleanResponse = strings.ReplaceAll(cleanResponse, call["original"], "")
		}

		// 构建包含工具结果的消息
		messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: cleanResponse})
		messages = append(messages, core.ChatMessage{
			Role:    core.RoleUser,
			Content: fmt.Sprintf("工具执行结果：\n%s\n\n请基于这些结果给出完整的回答。", strings.Join(toolResults, "\n\n")),
		})
	}

	// 如果超过最大迭代次数，获取最后一次回答
	if finalResponse == "" {
		response, err := a.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}
		finalResponse = response
	}

	// 保存到历史记录
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalResponse, core.RoleAssistant, core.Time{}, nil))

	return finalResponse, nil
}

// StreamRun 流式运行 Agent。
func (a *SimpleAgent) StreamRun(ctx context.Context, inputText string) (<-chan string, <-chan error) {
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.getEnhancedSystemPrompt()},
	}
	for _, msg := range a.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	streamCh, errCh := a.LLM.Think(ctx, messages, nil)

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

// AddTool 添加工具到 Agent。
func (a *SimpleAgent) AddTool(tool tools.Tool, autoExpand bool) {
	if a.toolRegistry == nil {
		a.toolRegistry = tools.NewToolRegistry()
		a.enableToolCalling = true
	}
	a.toolRegistry.RegisterTool(tool, autoExpand)
}

// RemoveTool 移除工具。
func (a *SimpleAgent) RemoveTool(toolName string) {
	if a.toolRegistry != nil {
		a.toolRegistry.Unregister(toolName)
	}
}

// ListTools 列出所有可用工具。
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
