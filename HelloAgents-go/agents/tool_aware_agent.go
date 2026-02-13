package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
)

// ToolCallListener 工具调用监听器回调函数类型
type ToolCallListener func(callInfo map[string]interface{})

// ToolAwareSimpleAgent SimpleAgent 子类，记录工具调用情况
type ToolAwareSimpleAgent struct {
	*SimpleAgent
	toolCallListener ToolCallListener
}

// NewToolAwareSimpleAgent 创建 ToolAwareSimpleAgent
func NewToolAwareSimpleAgent(
	name string,
	llm *core.HelloAgentsLLM,
	systemPrompt string,
	toolRegistry *tools.ToolRegistry,
	enableToolCalling bool,
	toolCallListener ToolCallListener,
) *ToolAwareSimpleAgent {
	return &ToolAwareSimpleAgent{
		SimpleAgent:      NewSimpleAgent(name, llm, systemPrompt, toolRegistry, enableToolCalling),
		toolCallListener: toolCallListener,
	}
}

// executeToolCallWithListener 执行工具调用并通知监听器
func (a *ToolAwareSimpleAgent) executeToolCallWithListener(toolName, parameters string) string {
	if a.SimpleAgent.toolRegistry == nil {
		return "❌ 错误：未配置工具注册表"
	}

	var parsedParameters map[string]interface{}
	var formattedResult string

	defer func() {
		// 通知监听器
		if a.toolCallListener != nil {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Tool call listener panicked: %v", r)
				}
			}()
			a.toolCallListener(map[string]interface{}{
				"agent_name":       a.Name,
				"tool_name":        toolName,
				"raw_parameters":   parameters,
				"parsed_parameters": parsedParameters,
				"result":           formattedResult,
			})
		}
	}()

	tool := a.SimpleAgent.toolRegistry.GetTool(toolName)
	if tool == nil {
		formattedResult = fmt.Sprintf("❌ 错误：未找到工具 '%s'", toolName)
		return formattedResult
	}

	parsedParameters = a.parseToolParametersWithSanitize(toolName, parameters)

	result, err := tool.Run(parsedParameters)
	if err != nil {
		formattedResult = fmt.Sprintf("❌ 工具调用失败：%s", err)
		return formattedResult
	}
	formattedResult = fmt.Sprintf("🔧 工具 %s 执行结果：\n%s", toolName, result)
	return formattedResult
}

// parseToolParametersWithSanitize 解析并清理工具参数
func (a *ToolAwareSimpleAgent) parseToolParametersWithSanitize(toolName, parameters string) map[string]interface{} {
	paramDict := a.SimpleAgent.parseToolParameters(toolName, parameters)
	return sanitizeParameters(paramDict)
}

// sanitizeParameters 清理和规范化工具参数
func sanitizeParameters(parameters map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range parameters {
		switch v := value.(type) {
		case int, float64, bool, []interface{}, map[string]interface{}:
			sanitized[key] = v
		case string:
			normalized := normalizeString(v)

			// 特殊处理 task_id
			if key == "task_id" {
				if i, err := strconv.Atoi(normalized); err == nil {
					sanitized[key] = i
					continue
				}
			}

			// 特殊处理 tags
			if key == "tags" {
				parsedTags := coerceSequence(normalized)
				if parsedTags != nil {
					sanitized[key] = parsedTags
					continue
				}
				if normalized != "" {
					tags := strings.Split(normalized, ",")
					cleanTags := make([]string, 0)
					for _, tag := range tags {
						tag = strings.TrimSpace(tag)
						if tag != "" {
							cleanTags = append(cleanTags, tag)
						}
					}
					if len(cleanTags) > 0 {
						sanitized[key] = cleanTags
						continue
					}
				}
			}

			sanitized[key] = normalized
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

// normalizeString 规范化字符串值，移除多余的引号和括号
func normalizeString(value string) string {
	trimmed := strings.TrimSpace(value)

	// 移除不成对的引号
	if len(trimmed) > 0 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '"' || first == '\'') && first != last {
			// 检查是否只有一个开头引号
			count := strings.Count(trimmed, string(first))
			if count == 1 {
				trimmed = trimmed[1:]
			}
		}
		if len(trimmed) > 0 {
			last = trimmed[len(trimmed)-1]
			if (last == '"' || last == '\'') && strings.Count(trimmed, string(last)) == 1 {
				trimmed = trimmed[:len(trimmed)-1]
			}
		}
	}

	// 移除成对的引号
	if len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '"' || first == '\'') && first == last {
			trimmed = trimmed[1 : len(trimmed)-1]
		}
	}

	// 补全不匹配的括号
	if len(trimmed) > 0 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if first == '[' && last != ']' {
			trimmed = trimmed + "]"
		} else if first == '(' && last != ')' {
			trimmed = trimmed + ")"
		}
	}

	return strings.TrimSpace(trimmed)
}

// coerceSequence 尝试将字符串转换为列表
func coerceSequence(value string) []interface{} {
	if value == "" {
		return nil
	}

	candidates := []string{value}
	if strings.HasPrefix(value, "[") && !strings.HasSuffix(value, "]") {
		candidates = append(candidates, value+"]")
	}
	if strings.HasPrefix(value, "(") && !strings.HasSuffix(value, ")") {
		candidates = append(candidates, value+")")
	}

	for _, candidate := range candidates {
		// 尝试 JSON 解析
		var parsed []interface{}
		if err := json.Unmarshal([]byte(candidate), &parsed); err == nil {
			return parsed
		}

		// 尝试 Python 列表解析
		if result := parsePythonListToInterface(candidate); result != nil {
			return result
		}
	}

	return nil
}

// parsePythonListToInterface 解析 Python 列表格式为 []interface{}
func parsePythonListToInterface(text string) []interface{} {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") {
		return nil
	}

	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	var result []interface{}
	var current strings.Builder
	inString := false
	stringQuote := byte(0)
	depth := 0

	for i := 0; i < len(text); i++ {
		char := text[i]

		if inString && char == '\\' && i+1 < len(text) {
			next := text[i+1]
			switch next {
			case 'n':
				current.WriteByte('\n')
			case 't':
				current.WriteByte('\t')
			case '\\':
				current.WriteByte('\\')
			case '"', '\'':
				current.WriteByte(next)
			default:
				current.WriteByte(char)
				current.WriteByte(next)
			}
			i++
			continue
		}

		if (char == '"' || char == '\'') && depth == 0 {
			if !inString {
				inString = true
				stringQuote = char
			} else if char == stringQuote {
				inString = false
				stringQuote = 0
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteByte(char)
			}
			continue
		}

		if inString {
			current.WriteByte(char)
			continue
		}

		if char == '[' || char == '{' {
			depth++
			current.WriteByte(char)
		} else if char == ']' || char == '}' {
			depth--
			current.WriteByte(char)
		} else if char == ',' && depth == 0 {
			continue
		} else if !isWhitespaceChar(char) {
			current.WriteByte(char)
		}
	}

	return result
}

// isWhitespaceChar 检查字节是否是空白字符
func isWhitespaceChar(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

// parseToolCallsEnhanced 增强的工具调用解析，支持嵌套结构
func (a *ToolAwareSimpleAgent) parseToolCallsEnhanced(text string) []map[string]string {
	marker := "[TOOL_CALL:"
	calls := make([]map[string]string, 0)
	start := 0

	for {
		begin := strings.Index(text[start:], marker)
		if begin == -1 {
			break
		}
		begin += start

		toolStart := begin + len(marker)
		colon := strings.Index(text[toolStart:], ":")
		if colon == -1 {
			break
		}
		colon += toolStart

		toolName := strings.TrimSpace(text[toolStart:colon])
		bodyStart := colon + 1
		pos := bodyStart
		depth := 0
		inString := false
		stringQuote := byte(0)

		for pos < len(text) {
			char := text[pos]

			// 处理字符串
			if char == '"' || char == '\'' {
				if !inString {
					inString = true
					stringQuote = char
				} else if stringQuote == char && (pos == 0 || text[pos-1] != '\\') {
					inString = false
				}
			}

			if !inString {
				if char == '[' {
					depth++
				} else if char == ']' {
					if depth == 0 {
						body := strings.TrimSpace(text[bodyStart:pos])
						original := text[begin : pos+1]
						calls = append(calls, map[string]string{
							"tool_name":  toolName,
							"parameters": body,
							"original":   original,
						})
						start = pos + 1
						break
					}
					depth--
				}
			}

			pos++
		}

		if pos >= len(text) {
			break
		}
	}

	return calls
}

// findToolCallEnd 查找工具调用的结束位置
func findToolCallEnd(text string, startIndex int) int {
	marker := "[TOOL_CALL:"
	toolStart := startIndex + len(marker)
	colon := strings.Index(text[toolStart:], ":")
	if colon == -1 {
		return -1
	}
	colon += toolStart

	bodyStart := colon + 1
	pos := bodyStart
	depth := 0
	inString := false
	stringQuote := byte(0)

	for pos < len(text) {
		char := text[pos]

		if char == '"' || char == '\'' {
			if !inString {
				inString = true
				stringQuote = char
			} else if stringQuote == char && (pos == 0 || text[pos-1] != '\\') {
				inString = false
			}
		}

		if !inString {
			if char == '[' {
				depth++
			} else if char == ']' {
				if depth == 0 {
					return pos
				}
				depth--
			}
		}

		pos++
	}

	return -1
}

// Run 运行 Agent（重写以使用监听器）
func (a *ToolAwareSimpleAgent) Run(ctx context.Context, inputText string, maxToolIterations int) (string, error) {
	if maxToolIterations <= 0 {
		maxToolIterations = 3
	}

	// 构建消息列表
	messages := []core.ChatMessage{
		{Role: core.RoleSystem, Content: a.SimpleAgent.getEnhancedSystemPrompt()},
	}

	// 添加历史消息
	for _, msg := range a.SimpleAgent.GetHistory() {
		messages = append(messages, msg.ToChatMessage())
	}

	// 添加当前用户消息
	messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

	// 如果没有启用工具调用，直接调用 LLM
	if !a.SimpleAgent.enableToolCalling {
		response, err := a.SimpleAgent.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}
		a.SimpleAgent.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
		a.SimpleAgent.AddMessage(core.NewMessage(response, core.RoleAssistant, core.Time{}, nil))
		return response, nil
	}

	// 迭代处理，支持多轮工具调用
	var finalResponse string
	for i := 0; i < maxToolIterations; i++ {
		response, err := a.SimpleAgent.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		toolCalls := a.parseToolCallsEnhanced(response)
		if len(toolCalls) == 0 {
			finalResponse = response
			break
		}

		// 执行所有工具调用
		var toolResults []string
		cleanResponse := response
		for _, call := range toolCalls {
			result := a.executeToolCallWithListener(call["tool_name"], call["parameters"])
			toolResults = append(toolResults, result)
			cleanResponse = strings.Replace(cleanResponse, call["original"], "", 1)
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
		response, err := a.SimpleAgent.LLM.Invoke(ctx, messages, nil)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}
		finalResponse = response
	}

	// 保存到历史记录
	a.SimpleAgent.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
	a.SimpleAgent.AddMessage(core.NewMessage(finalResponse, core.RoleAssistant, core.Time{}, nil))

	return finalResponse, nil
}

// StreamRun 流式运行 Agent，支持工具调用
func (a *ToolAwareSimpleAgent) StreamRun(ctx context.Context, inputText string, maxToolIterations int) (<-chan string, <-chan error) {
	outCh := make(chan string, 64)
	outErrCh := make(chan error, 1)

	go func() {
		defer close(outCh)
		defer close(outErrCh)

		if maxToolIterations <= 0 {
			maxToolIterations = 3
		}

		// 构建消息列表
		messages := []core.ChatMessage{
			{Role: core.RoleSystem, Content: a.SimpleAgent.getEnhancedSystemPrompt()},
		}

		// 添加历史消息
		for _, msg := range a.SimpleAgent.GetHistory() {
			messages = append(messages, msg.ToChatMessage())
		}

		// 添加当前用户消息
		messages = append(messages, core.ChatMessage{Role: core.RoleUser, Content: inputText})

		// 如果没有启用工具调用，直接流式调用 LLM
		if !a.SimpleAgent.enableToolCalling {
			streamCh, errCh := a.SimpleAgent.LLM.Think(ctx, messages, nil)

			var fullResponse strings.Builder
			for {
				select {
				case <-ctx.Done():
					outErrCh <- ctx.Err()
					return
				case chunk, ok := <-streamCh:
					if !ok {
						a.SimpleAgent.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
						a.SimpleAgent.AddMessage(core.NewMessage(fullResponse.String(), core.RoleAssistant, core.Time{}, nil))
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
		}

		// 流式工具调用支持
		var finalSegments []string
		var finalResponseText string
		currentIteration := 0
		marker := "[TOOL_CALL:"

		for currentIteration < maxToolIterations {
			residual := ""
			var segmentsThisRound []string
			var toolCallTexts []string

			streamCh, errCh := a.SimpleAgent.LLM.Think(ctx, messages, nil)

			// 处理流式输出
			processResidual := func(finalPass bool) {
				for {
					start := strings.Index(residual, marker)
					if start == -1 {
						safeLen := len(residual)
						if !finalPass {
							safeLen = max(0, len(residual)-(len(marker)-1))
						}
						if safeLen > 0 {
							segment := residual[:safeLen]
							residual = residual[safeLen:]
							if segment != "" {
								segmentsThisRound = append(segmentsThisRound, segment)
								finalSegments = append(finalSegments, segment)
								outCh <- segment
							}
						}
						break
					}

					if start > 0 {
						segment := residual[:start]
						residual = residual[start:]
						if segment != "" {
							segmentsThisRound = append(segmentsThisRound, segment)
							finalSegments = append(finalSegments, segment)
							outCh <- segment
						}
						continue
					}

					end := findToolCallEnd(residual, 0)
					if end == -1 {
						break
					}

					toolCallTexts = append(toolCallTexts, residual[:end+1])
					residual = residual[end+1:]
				}
			}

		streamLoop:
			for {
				select {
				case <-ctx.Done():
					outErrCh <- ctx.Err()
					return
				case chunk, ok := <-streamCh:
					if !ok {
						break streamLoop
					}
					if chunk == "" {
						continue
					}
					residual += chunk

					// 处理残留在缓冲区中的内容
					for {
						start := strings.Index(residual, marker)
						if start == -1 {
							// 没有找到工具调用标记，输出安全部分
							safeLen := max(0, len(residual)-(len(marker)-1))
							if safeLen > 0 {
								segment := residual[:safeLen]
								residual = residual[safeLen:]
								if segment != "" {
									segmentsThisRound = append(segmentsThisRound, segment)
									finalSegments = append(finalSegments, segment)
									outCh <- segment
								}
							}
							break
						}

						if start > 0 {
							segment := residual[:start]
							residual = residual[start:]
							if segment != "" {
								segmentsThisRound = append(segmentsThisRound, segment)
								finalSegments = append(finalSegments, segment)
								outCh <- segment
							}
							continue
						}

						// 找到工具调用，查找结束位置
						end := findToolCallEnd(residual, 0)
						if end == -1 {
							// 没有找到结束位置，等待更多数据
							break
						}

						toolCallTexts = append(toolCallTexts, residual[:end+1])
						residual = residual[end+1:]
					}

				case err, ok := <-errCh:
					if ok && err != nil {
						outErrCh <- err
						return
					}
					break streamLoop
				}
			}

			// 处理最终残留
			processResidual(true)

			cleanResponse := strings.Join(segmentsThisRound, "")
			var toolCalls []map[string]string

			// 解析工具调用
			for _, callText := range toolCallTexts {
				toolCalls = append(toolCalls, a.parseToolCallsEnhanced(callText)...)
			}

			if len(toolCalls) > 0 {
				messages = append(messages, core.ChatMessage{Role: core.RoleAssistant, Content: cleanResponse})

				var toolResults []string
				for _, call := range toolCalls {
					result := a.executeToolCallWithListener(call["tool_name"], call["parameters"])
					toolResults = append(toolResults, result)
				}

				toolResultsText := strings.Join(toolResults, "\n\n")
				messages = append(messages, core.ChatMessage{
					Role:    core.RoleUser,
					Content: fmt.Sprintf("工具执行结果：\n%s\n\n请基于这些结果给出完整的回答。", toolResultsText),
				})

				currentIteration++
				continue
			}

			finalResponseText = cleanResponse
			break
		}

		// 如果超过最大迭代次数，获取最后一次回答
		if currentIteration >= maxToolIterations && finalResponseText == "" {
			response, err := a.SimpleAgent.LLM.Invoke(ctx, messages, nil)
			if err != nil {
				outErrCh <- err
				return
			}
			finalSegments = append(finalSegments, response)
			finalResponseText = response
			outCh <- response
		}

		// 保存到历史记录
		storedResponse := finalResponseText
		if storedResponse == "" {
			storedResponse = strings.Join(finalSegments, "")
		}
		a.SimpleAgent.AddMessage(core.NewMessage(inputText, core.RoleUser, core.Time{}, nil))
		a.SimpleAgent.AddMessage(core.NewMessage(storedResponse, core.RoleAssistant, core.Time{}, nil))
	}()

	return outCh, outErrCh
}

// AttachRegistry 附加工具注册表
func AttachRegistry(agent *ToolAwareSimpleAgent, registry *tools.ToolRegistry) {
	if registry != nil {
		agent.SimpleAgent.toolRegistry = registry
		agent.SimpleAgent.enableToolCalling = true
	}
}

// max 返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Compile-time assertion to ensure ToolAwareSimpleAgent implements expected interfaces
var _ fmt.Stringer = (*ToolAwareSimpleAgent)(nil)

// String 返回字符串表示
func (a *ToolAwareSimpleAgent) String() string {
	return fmt.Sprintf("ToolAwareSimpleAgent(name=%s)", a.Name)
}

// parsePythonList 解析 Python 列表格式（复用已有实现）
var _ = regexp.MustCompile("") // 确保 regexp 包被导入
