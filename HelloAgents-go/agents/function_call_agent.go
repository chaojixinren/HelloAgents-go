package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"

	openai "github.com/sashabaranov/go-openai"
)

// mapParameterType 将工具参数类型映射为 JSON Schema 允许的类型
func mapParameterType(paramType string) string {
	normalized := strings.ToLower(paramType)
	switch normalized {
	case "string", "number", "integer", "boolean", "array", "object":
		return normalized
	default:
		return "string"
	}
}

// FunctionCallAgent 基于 OpenAI 原生函数调用机制的 Agent
type FunctionCallAgent struct {
	*core.BaseAgent
	toolRegistry        *tools.ToolRegistry
	enableToolCalling   bool
	defaultToolChoice   string
	maxToolIterations   int
}

// NewFunctionCallAgent 创建 FunctionCallAgent
func NewFunctionCallAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	config *core.Config,
	toolRegistry *tools.ToolRegistry,
	enableToolCalling bool,
	defaultToolChoice string,
	maxToolIterations int,
) *FunctionCallAgent {
	if defaultToolChoice == "" {
		defaultToolChoice = "auto"
	}
	if maxToolIterations <= 0 {
		maxToolIterations = 3
	}

	return &FunctionCallAgent{
		BaseAgent:         core.NewBaseAgent(name, llm, systemPrompt, config),
		toolRegistry:      toolRegistry,
		enableToolCalling: enableToolCalling && toolRegistry != nil,
		defaultToolChoice: defaultToolChoice,
		maxToolIterations: maxToolIterations,
	}
}

// getSystemPrompt 构建系统提示词，注入工具描述
func (a *FunctionCallAgent) getSystemPrompt() string {
	basePrompt := a.SystemPrompt
	if basePrompt == "" {
		basePrompt = "你是一个可靠的AI助理，能够在需要时调用工具完成任务。"
	}

	if !a.enableToolCalling || a.toolRegistry == nil {
		return basePrompt
	}

	toolsDescription := a.toolRegistry.GetToolsDescription()
	if toolsDescription == "" || toolsDescription == "暂无可用工具" {
		return basePrompt
	}

	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n## 可用工具\n")
	sb.WriteString("当你判断需要外部信息或执行动作时，可以直接通过函数调用使用以下工具：\n")
	sb.WriteString(toolsDescription)
	sb.WriteString("\n\n请主动决定是否调用工具，合理利用多次调用来获得完备答案。")

	return sb.String()
}

// functionInfo 用于存储从反射中提取的函数信息
type functionInfo struct {
	description string
}

// getClientViaReflection 通过反射获取 HelloAgentsLLM 中的 client 字段
func (a *FunctionCallAgent) getClientViaReflection() *openai.Client {
	if a.LLM == nil {
		return nil
	}

	// 使用反射访问私有字段
	v := reflect.ValueOf(a.LLM).Elem()
	clientField := v.FieldByName("client")

	if !clientField.IsValid() {
		return nil
	}

	// 获取指针值
	if clientField.Kind() == reflect.Interface || clientField.Kind() == reflect.Ptr {
		if clientField.IsNil() {
			return nil
		}
		// 转换为 *openai.Client
		if c, ok := clientField.Interface().(*openai.Client); ok {
			return c
		}
	}

	return nil
}

// getFunctionsViaReflection 通过反射获取 ToolRegistry 中的 _functions 字段
func (a *FunctionCallAgent) getFunctionsViaReflection() map[string]functionInfo {
	result := make(map[string]functionInfo)

	if a.toolRegistry == nil {
		return result
	}

	// 使用反射访问私有字段
	v := reflect.ValueOf(a.toolRegistry).Elem()
	functionsField := v.FieldByName("_functions")

	if !functionsField.IsValid() {
		return result
	}

	// 遍历 map
	iter := functionsField.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		value := iter.Value()

		// 获取 functionInfo 结构体的 description 字段
		descriptionField := value.FieldByName("description")
		if descriptionField.IsValid() {
			result[key] = functionInfo{
				description: descriptionField.String(),
			}
		}
	}

	return result
}

// buildToolSchemas 构建工具的 JSON Schema
func (a *FunctionCallAgent) buildToolSchemas() []openai.Tool {
	if !a.enableToolCalling || a.toolRegistry == nil {
		return nil
	}

	schemas := make([]openai.Tool, 0)

	// Tool 对象
	for _, tool := range a.toolRegistry.GetAllTools() {
		properties := make(map[string]openai.JSONSchemaDefinition)
		required := make([]string, 0)

		parameters := tool.GetParameters()
		for _, param := range parameters {
			properties[param.Name] = openai.JSONSchemaDefinition{
				Type:        mapParameterType(param.Type),
				Description: param.Description,
			}
			if param.Default != nil {
				// OpenAI schema 不支持 default 字段，添加到描述中
				properties[param.Name] = openai.JSONSchemaDefinition{
					Type:        mapParameterType(param.Type),
					Description: fmt.Sprintf("%s (默认: %v)", param.Description, param.Default),
				}
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}

		schema := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters: &openai.FunctionParameters{
					"type":       "object",
					"properties": properties,
					"required":   required,
				},
			},
		}
		schemas = append(schemas, schema)
	}

	// register_function 注册的工具（通过反射访问内部结构）
	functionMap := a.getFunctionsViaReflection()
	for name, info := range functionMap {
		schema := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: info.description,
				Parameters: &openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{
							"type":        "string",
							"description": "输入文本",
						},
					},
					"required": []string{"input"},
				},
			},
		}
		schemas = append(schemas, schema)
	}

	return schemas
}

// parseFunctionCallArguments 解析模型返回的 JSON 字符串参数
func (a *FunctionCallAgent) parseFunctionCallArguments(arguments string) map[string]interface{} {
	if arguments == "" {
		return make(map[string]interface{})
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
		return make(map[string]interface{})
	}
	return parsed
}

// convertParameterTypes 根据工具定义尽可能转换参数类型
func (a *FunctionCallAgent) convertParameterTypes(toolName string, paramDict map[string]interface{}) map[string]interface{} {
	if a.toolRegistry == nil {
		return paramDict
	}

	tool := a.toolRegistry.GetTool(toolName)
	if tool == nil {
		return paramDict
	}

	toolParams := tool.GetParameters()
	typeMapping := make(map[string]string)
	for _, param := range toolParams {
		typeMapping[param.Name] = param.Type
	}

	converted := make(map[string]interface{})
	for key, value := range paramDict {
		paramType, ok := typeMapping[key]
		if !ok {
			converted[key] = value
			continue
		}

		normalized := strings.ToLower(paramType)
		switch normalized {
		case "number", "float":
			if f, ok := toFloat64(value); ok {
				converted[key] = f
			} else {
				converted[key] = value
			}
		case "integer", "int":
			if i, ok := toInt(value); ok {
				converted[key] = i
			} else {
				converted[key] = value
			}
		case "boolean", "bool":
			converted[key] = toBool(value)
		default:
			converted[key] = value
		}
	}

	return converted
}

// toFloat64 尝试将值转换为 float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// toInt 尝试将值转换为 int
func toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}

// toBool 将值转换为 bool
func toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int, int64, float64:
		return v != 0
	case string:
		lowered := strings.ToLower(v)
		return lowered == "true" || lowered == "1" || lowered == "yes"
	default:
		return false
	}
}

// executeToolCall 执行工具调用并返回字符串结果
func (a *FunctionCallAgent) executeToolCall(toolName string, arguments map[string]interface{}) string {
	if a.toolRegistry == nil {
		return "❌ 错误：未配置工具注册表"
	}

	tool := a.toolRegistry.GetTool(toolName)
	if tool != nil {
		typedArguments := a.convertParameterTypes(toolName, arguments)
		result, err := tool.Run(typedArguments)
		if err != nil {
			return fmt.Sprintf("❌ 工具调用失败：%s", err)
		}
		return result
	}

	// 尝试获取函数工具
	fn := a.toolRegistry.GetFunction(toolName)
	if fn != nil {
		inputText := ""
		if v, ok := arguments["input"]; ok {
			if s, ok := v.(string); ok {
				inputText = s
			}
		}
		return fn(inputText)
	}

	return fmt.Sprintf("❌ 错误：未找到工具 '%s'", toolName)
}

// Run 执行函数调用范式的对话流程
func (a *FunctionCallAgent) Run(ctx context.Context, inputText string) (string, error) {
	return a.RunWithOptions(ctx, inputText, 0, "")
}

// RunWithOptions 执行函数调用范式的对话流程（带选项）
func (a *FunctionCallAgent) RunWithOptions(ctx context.Context, inputText string, maxToolIterations int, toolChoice string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)

	// 添加系统提示词
	systemPrompt := a.getSystemPrompt()
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})

	// 添加历史消息
	for _, msg := range a.GetHistory() {
		role := openai.ChatMessageRoleUser
		switch strings.ToLower(msg.Role) {
		case "system":
			role = openai.ChatMessageRoleSystem
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		case "user":
			role = openai.ChatMessageRoleUser
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// 添加当前用户消息
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: inputText,
	})

	toolSchemas := a.buildToolSchemas()
	if len(toolSchemas) == 0 {
		// 没有工具，直接调用 LLM
		chatMessages := a.convertToChatMessages(messages)
		response, err := a.LLM.Invoke(ctx, chatMessages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}
		a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
		a.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))
		return response, nil
	}

	iterationsLimit := maxToolIterations
	if iterationsLimit <= 0 {
		iterationsLimit = a.maxToolIterations
	}
	effectiveToolChoice := toolChoice
	if effectiveToolChoice == "" {
		effectiveToolChoice = a.defaultToolChoice
	}

	currentIteration := 0
	finalResponse := ""

	for currentIteration < iterationsLimit {
		req := openai.ChatCompletionRequest{
			Model:    a.LLM.Model,
			Messages: messages,
			Tools:    toolSchemas,
		}

		// 设置 tool_choice
		switch effectiveToolChoice {
		case "auto":
			req.ToolChoice = "auto"
		case "none":
			req.ToolChoice = "none"
		case "required":
			req.ToolChoice = "required"
		default:
			req.ToolChoice = "auto"
		}

		if a.LLM.MaxTokens > 0 {
			req.MaxTokens = a.LLM.MaxTokens
		}

		if ctx == nil {
			ctx = context.Background()
		}

		response, err := a.getClientViaReflection().CreateChatCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		if len(response.Choices) == 0 {
			return "", fmt.Errorf("LLM 返回无内容")
		}

		choice := response.Choices[0]
		assistantMessage := choice.Message
		content := assistantMessage.Content
		toolCalls := assistantMessage.ToolCalls

		if len(toolCalls) > 0 {
			// 构建助手消息（包含工具调用）
			assistantMsg := openai.ChatCompletionMessage{
				Role:         openai.ChatMessageRoleAssistant,
				Content:      content,
				ToolCalls:    toolCalls,
			}
			messages = append(messages, assistantMsg)

			// 执行每个工具调用
			for _, toolCall := range toolCalls {
				toolName := toolCall.Function.Name
				arguments := a.parseFunctionCallArguments(toolCall.Function.Arguments)
				result := a.executeToolCall(toolName, arguments)

				// 添加工具结果消息
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}

			currentIteration++
			continue
		}

		finalResponse = content
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: finalResponse,
		})
		break
	}

	// 如果超过最大迭代次数，获取最后一次回答
	if currentIteration >= iterationsLimit && finalResponse == "" {
		req := openai.ChatCompletionRequest{
			Model:    a.LLM.Model,
			Messages: messages,
			Tools:    toolSchemas,
		}
		req.ToolChoice = "none"

		if a.LLM.MaxTokens > 0 {
			req.MaxTokens = a.LLM.MaxTokens
		}

		response, err := a.getClientViaReflection().CreateChatCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		if len(response.Choices) > 0 {
			finalResponse = response.Choices[0].Message.Content
		}
	}

	// 保存到历史记录
	a.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.AddMessage(core.NewMessage(finalResponse, core.RoleAssistant, core.Time{}, nil))

	return finalResponse, nil
}

// convertToChatMessages 将 openai.ChatCompletionMessage 转换为 core.ChatMessage
func (a *FunctionCallAgent) convertToChatMessages(messages []openai.ChatCompletionMessage) []core.ChatMessage {
	result := make([]core.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		role := core.RoleUser
		switch msg.Role {
		case openai.ChatMessageRoleSystem:
			role = core.RoleSystem
		case openai.ChatMessageRoleAssistant:
			role = core.RoleAssistant
		case openai.ChatMessageRoleUser:
			role = core.RoleUser
		}
		result = append(result, core.ChatMessage{Role: role, Content: msg.Content})
	}
	return result
}

// AddTool 添加工具到 Agent
func (a *FunctionCallAgent) AddTool(tool tools.Tool, autoExpand bool) {
	if a.toolRegistry == nil {
		a.toolRegistry = tools.NewToolRegistry()
		a.enableToolCalling = true
	}
	a.toolRegistry.RegisterTool(tool, autoExpand)
}

// RemoveTool 移除工具
func (a *FunctionCallAgent) RemoveTool(toolName string) bool {
	if a.toolRegistry != nil {
		before := a.toolRegistry.ListTools()
		a.toolRegistry.Unregister(toolName)
		after := a.toolRegistry.ListTools()
		return contains(before, toolName) && !contains(after, toolName)
	}
	return false
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ListTools 列出所有可用工具
func (a *FunctionCallAgent) ListTools() []string {
	if a.toolRegistry != nil {
		return a.toolRegistry.ListTools()
	}
	return []string{}
}

// HasTools 检查是否有可用工具
func (a *FunctionCallAgent) HasTools() bool {
	return a.enableToolCalling && a.toolRegistry != nil
}
