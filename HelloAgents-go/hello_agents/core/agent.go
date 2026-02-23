package core

import (
	stdctx "context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	haContext "helloagents-go/hello_agents/context"
	"helloagents-go/hello_agents/observability"
	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

// Agent defines common behavior all agents should expose.
type Agent interface {
	Run(inputText string, kwargs map[string]any) (string, error)
	Arun(inputText string, hooks Hooks, kwargs map[string]any) (string, error)
	ArunStream(inputText string, kwargs map[string]any) <-chan AgentEvent
	AddMessage(message Message)
	ClearHistory()
	GetHistory() []Message
}

// Hooks mirrors lifecycle callback set in Python arun API.
type Hooks struct {
	OnStart    LifecycleHook
	OnStep     LifecycleHook
	OnToolCall LifecycleHook
	OnFinish   LifecycleHook
	OnError    LifecycleHook
}

// BaseAgent is the scaffold counterpart of Python Agent base class.
type BaseAgent struct {
	Name         string
	LLM          *HelloAgentsLLM
	SystemPrompt string
	Config       Config
	AgentType    string
	MaxSteps     int

	ToolRegistry *tools.ToolRegistry

	HistoryManager *haContext.HistoryManager[Message]
	Truncator      *haContext.ObservationTruncator
	TokenCounter   *haContext.TokenCounter[Message]

	TraceLogger  *observability.TraceLogger
	SkillLoader  *skills.SkillLoader
	SessionStore *SessionStore
	summaryLLM   *HelloAgentsLLM

	HistoryTokenCount int
	SessionMetadata   map[string]any
	StartTime         time.Time
}

func NewBaseAgent(name string, llm *HelloAgentsLLM, systemPrompt string, config *Config, toolRegistry *tools.ToolRegistry) (*BaseAgent, error) {
	cfg := DefaultConfig()
	if config != nil {
		cfg = *config
	}

	historyManager := haContext.NewHistoryManager[Message](
		cfg.MinRetainRounds,
		cfg.CompressionThreshold,
		func(summary string) Message {
			return NewMessage(
				"## Archived Session Summary\n"+summary,
				MessageRoleSummary,
				map[string]any{"compressed_at": time.Now().Format(time.RFC3339Nano)},
			)
		},
		func(msg Message) string {
			return string(msg.Role)
		},
	)
	truncator := haContext.NewObservationTruncator(cfg.ToolOutputMaxLines, cfg.ToolOutputMaxBytes, cfg.ToolOutputTruncateDirection, cfg.ToolOutputDir)
	tokenCounter := haContext.NewTokenCounter[Message](
		llm.Model,
		func(msg Message) string {
			return msg.Content
		},
		func(msg Message) string {
			return string(msg.Role) + ":" + msg.Content
		},
	)

	agent := &BaseAgent{
		Name:           name,
		LLM:            llm,
		SystemPrompt:   systemPrompt,
		Config:         cfg,
		AgentType:      "BaseAgent",
		ToolRegistry:   toolRegistry,
		HistoryManager: historyManager,
		Truncator:      truncator,
		TokenCounter:   tokenCounter,
		StartTime:      time.Now(),
		SessionMetadata: map[string]any{
			"created_at":       time.Now().Format(time.RFC3339Nano),
			"total_tokens":     0,
			"total_steps":      0,
			"duration_seconds": 0,
		},
	}

	if cfg.TraceEnabled {
		traceLogger, err := observability.NewTraceLogger(cfg.TraceDir, cfg.TraceSanitize, cfg.TraceHTMLIncludeRawResponse)
		if err == nil {
			agent.TraceLogger = traceLogger
			agent.TraceLogger.LogEvent("session_start", map[string]any{
				"agent_name": name,
				"agent_type": "BaseAgent",
				"config":     cfg.ToMap(),
			}, nil)
		}
	}

	if cfg.SkillsEnabled {
		loader, err := skills.NewSkillLoader(cfg.SkillsDir)
		if err == nil {
			agent.SkillLoader = loader
			if cfg.SkillsAutoRegister && toolRegistry != nil {
				toolRegistry.RegisterTool(builtin.NewSkillTool(loader), false)
			}
		}
	}

	if cfg.SessionEnabled {
		store, err := NewSessionStore(cfg.SessionDir)
		if err == nil {
			agent.SessionStore = store
		}
	}

	return agent, nil
}

func (a *BaseAgent) Run(inputText string, kwargs map[string]any) (string, error) {
	return "", fmt.Errorf("run() not implemented in BaseAgent")
}

func (a *BaseAgent) Arun(inputText string, hooks Hooks, kwargs map[string]any) (string, error) {
	if err := a.emitEvent(AgentStart, hooks.OnStart, map[string]any{"input_text": inputText}); err != nil {
		return "", err
	}

	result, err := a.Run(inputText, kwargs)
	if err != nil {
		_ = a.emitEvent(AgentError, hooks.OnError, map[string]any{"error": err.Error(), "error_type": "AgentError"})
		return "", err
	}

	_ = a.emitEvent(AgentFinish, hooks.OnFinish, map[string]any{"result": result})
	return result, nil
}

func (a *BaseAgent) ArunStream(inputText string, kwargs map[string]any) <-chan AgentEvent {
	out := make(chan AgentEvent, 2)
	go func() {
		defer close(out)
		out <- NewAgentEvent(AgentStart, a.Name, map[string]any{"input_text": inputText})
		result, err := a.Run(inputText, kwargs)
		if err != nil {
			out <- NewAgentEvent(AgentError, a.Name, map[string]any{"error": err.Error(), "error_type": "AgentError"})
			return
		}
		out <- NewAgentEvent(AgentFinish, a.Name, map[string]any{"result": result})
	}()
	return out
}

func (a *BaseAgent) emitEvent(eventType EventType, hook LifecycleHook, data map[string]any) error {
	event := NewAgentEvent(eventType, a.Name, data)
	if hook == nil {
		return nil
	}
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), time.Duration(a.Config.HookTimeoutSeconds*float64(time.Second)))
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- hook(event)
	}()

	select {
	case <-ctx.Done():
		if a.TraceLogger != nil {
			a.TraceLogger.LogEvent("hook_timeout", map[string]any{"event_type": string(eventType), "timeout": a.Config.HookTimeoutSeconds}, nil)
		}
		return nil
	case err := <-done:
		if err != nil && a.TraceLogger != nil {
			a.TraceLogger.LogEvent("hook_error", map[string]any{"event_type": string(eventType), "error": err.Error()}, nil)
		}
		return nil
	}
}

func (a *BaseAgent) AddMessage(message Message) {
	a.HistoryManager.Append(message)
	a.HistoryTokenCount += a.TokenCounter.CountMessage(message)

	if a.ShouldCompress() {
		a.CompressHistory()
	}

	if a.Config.AutoSaveEnabled && a.SessionStore != nil {
		historyLen := len(a.HistoryManager.GetHistory())
		if a.Config.AutoSaveInterval > 0 && historyLen%a.Config.AutoSaveInterval == 0 {
			a.AutoSave()
		}
	}
}

func (a *BaseAgent) ClearHistory() {
	a.HistoryManager.Clear()
	a.HistoryTokenCount = 0
	a.TokenCounter.ClearCache()
}

func (a *BaseAgent) GetHistory() []Message {
	return a.HistoryManager.GetHistory()
}

func (a *BaseAgent) ShouldCompress() bool {
	threshold := int(float64(a.Config.ContextWindow) * a.Config.CompressionThreshold)
	return a.HistoryTokenCount > threshold
}

func (a *BaseAgent) CompressHistory() {
	history := a.HistoryManager.GetHistory()
	summary := a.GenerateSimpleSummary(history)
	if a.Config.EnableSmartCompression {
		summary = a.GenerateSmartSummary(history)
	}
	a.HistoryManager.Compress(summary)
	a.HistoryTokenCount = a.TokenCounter.CountMessages(a.HistoryManager.GetHistory())
}

func (a *BaseAgent) GenerateSimpleSummary(history []Message) string {
	rounds := a.HistoryManager.EstimateRounds()
	userMsgs := 0
	assistantMsgs := 0
	for _, msg := range history {
		if msg.Role == MessageRoleUser {
			userMsgs++
		}
		if msg.Role == MessageRoleAssistant {
			assistantMsgs++
		}
	}
	return fmt.Sprintf("此会话包含 %d 轮对话：\n- 用户消息：%d 条\n- 助手消息：%d 条\n- 总消息数：%d 条\n\n（历史已压缩，保留最近 %d 轮完整对话）", rounds, userMsgs, assistantMsgs, len(history), a.Config.MinRetainRounds)
}

func (a *BaseAgent) GenerateSmartSummary(history []Message) string {
	boundaries := a.HistoryManager.FindRoundBoundaries()
	if len(boundaries) <= a.Config.MinRetainRounds {
		return a.GenerateSimpleSummary(history)
	}

	keepFromIndex := boundaries[len(boundaries)-a.Config.MinRetainRounds]
	if keepFromIndex <= 0 || keepFromIndex > len(history) {
		return a.GenerateSimpleSummary(history)
	}

	toCompress := history[:keepFromIndex]
	if len(toCompress) == 0 {
		return a.GenerateSimpleSummary(history)
	}

	historyText := a.FormatHistoryForSummary(toCompress)
	summaryPrompt := fmt.Sprintf(`请将以下对话历史压缩为结构化摘要，保留关键信息：

## 对话历史
%s

## 摘要要求
1. **任务目标**：用户想要完成什么？
2. **关键决策**：做了哪些重要决定？
3. **已完成工作**：完成了哪些任务？（列表形式）
4. **待处理事项**：还有什么未完成？
5. **重要发现**：有哪些关键信息或问题？

请用简洁的中文输出，每部分不超过 3 行。`, historyText)

	summaryLLM, err := a.getSummaryLLM()
	if err != nil {
		if a.Config.Debug {
			fmt.Printf("⚠️ 智能摘要生成失败: %v，使用简单摘要\n", err)
		}
		return a.GenerateSimpleSummary(history)
	}

	response, err := summaryLLM.Invoke(
		[]map[string]any{
			{"role": "system", "content": "你是一个专业的对话摘要助手，擅长提取关键信息。"},
			{"role": "user", "content": summaryPrompt},
		},
		map[string]any{
			"temperature": a.Config.SummaryTemperature,
			"max_tokens":  a.Config.SummaryMaxTokens,
		},
	)
	if err != nil {
		if a.Config.Debug {
			fmt.Printf("⚠️ 智能摘要生成失败: %v，使用简单摘要\n", err)
		}
		return a.GenerateSimpleSummary(history)
	}

	return fmt.Sprintf(`## 历史摘要（%d 条消息）
%s

---
（已压缩，保留最近 %d 轮完整对话）`,
		len(toCompress),
		response.Content,
		a.Config.MinRetainRounds,
	)
}

func (a *BaseAgent) FormatHistoryForSummary(history []Message) string {
	lines := make([]string, 0, len(history))
	for _, msg := range history {
		content := msg.Content
		if len(content) > 500 {
			content = content[:500]
		}
		lines = append(lines, fmt.Sprintf("[%s]: %s", msg.Role, content))
	}
	return strings.Join(lines, "\n\n")
}

func (a *BaseAgent) getSummaryLLM() (*HelloAgentsLLM, error) {
	if a.summaryLLM != nil {
		return a.summaryLLM, nil
	}

	if a.LLM == nil {
		return nil, fmt.Errorf("llm is not initialized")
	}

	model := strings.TrimSpace(a.Config.SummaryLLMModel)
	if model == "" {
		model = a.LLM.Model
	}

	maxTokens := a.Config.SummaryMaxTokens
	timeout := a.LLM.Timeout

	llm, err := NewHelloAgentsLLM(
		model,
		a.LLM.APIKey,
		a.LLM.BaseURL,
		a.Config.SummaryTemperature,
		&maxTokens,
		&timeout,
		map[string]any{"provider": a.Config.SummaryLLMProvider},
	)
	if err != nil {
		return nil, err
	}

	a.summaryLLM = llm
	return a.summaryLLM, nil
}

func mapParameterType(paramType string) string {
	normalized := strings.ToLower(strings.TrimSpace(paramType))
	switch normalized {
	case "string", "number", "integer", "boolean", "array", "object":
		return normalized
	default:
		return "string"
	}
}

func convertParameterTypes(params []tools.ToolParameter, input map[string]any) map[string]any {
	typeMap := map[string]string{}
	for _, p := range params {
		typeMap[p.Name] = strings.ToLower(strings.TrimSpace(p.Type))
	}
	converted := map[string]any{}
	for key, value := range input {
		targetType := typeMap[key]
		switch targetType {
		case "number", "float":
			switch v := value.(type) {
			case float64:
				converted[key] = v
			case float32:
				converted[key] = float64(v)
			case int:
				converted[key] = float64(v)
			case int64:
				converted[key] = float64(v)
			case string:
				var parsed float64
				if _, err := fmt.Sscanf(v, "%f", &parsed); err == nil {
					converted[key] = parsed
				} else {
					converted[key] = value
				}
			default:
				converted[key] = value
			}
		case "integer", "int":
			switch v := value.(type) {
			case int:
				converted[key] = v
			case int64:
				converted[key] = int(v)
			case float64:
				converted[key] = int(v)
			case string:
				var parsed int
				if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
					converted[key] = parsed
				} else {
					converted[key] = value
				}
			default:
				converted[key] = value
			}
		case "boolean", "bool":
			switch v := value.(type) {
			case bool:
				converted[key] = v
			case string:
				lower := strings.ToLower(strings.TrimSpace(v))
				converted[key] = lower == "true" || lower == "1" || lower == "yes"
			case int:
				converted[key] = v != 0
			default:
				converted[key] = value
			}
		default:
			converted[key] = value
		}
	}
	return converted
}

func (a *BaseAgent) BuildToolSchemas() []map[string]any {
	if a.ToolRegistry == nil {
		return []map[string]any{}
	}

	schemas := make([]map[string]any, 0)

	// Tool object schemas.
	for _, tool := range a.ToolRegistry.GetAllTools() {
		properties := map[string]any{}
		required := make([]string, 0)

		for _, param := range tool.GetParameters() {
			prop := map[string]any{
				"type":        mapParameterType(param.Type),
				"description": param.Description,
			}
			if param.Default != nil {
				prop["default"] = param.Default
			}
			properties[param.Name] = prop
			if param.Required {
				required = append(required, param.Name)
			}
		}

		parameters := map[string]any{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			parameters["required"] = required
		}

		schema := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.GetName(),
				"description": tool.GetDescription(),
				"parameters":  parameters,
			},
		}
		schemas = append(schemas, schema)
	}

	// Function tools.
	for name, info := range a.ToolRegistry.GetAllFunctions() {
		schemas = append(schemas, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": info.Description,
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{
							"type":        "string",
							"description": "输入文本",
						},
					},
					"required": []string{"input"},
				},
			},
		})
	}

	return schemas
}

func (a *BaseAgent) ExecuteToolCall(toolName string, arguments map[string]any) string {
	if a.ToolRegistry == nil {
		return "❌ 错误：未配置工具注册表"
	}

	// 1) Tool object
	if tool := a.ToolRegistry.GetTool(toolName); tool != nil {
		typedArguments := convertParameterTypes(tool.GetParameters(), arguments)
		response := tool.RunWithTiming(typedArguments)

		switch response.Status {
		case tools.ToolStatusError:
			code := "UNKNOWN"
			if response.ErrorInfo != nil && response.ErrorInfo["code"] != "" {
				code = response.ErrorInfo["code"]
			}
			return fmt.Sprintf("❌ 错误 [%s]: %s", code, response.Text)
		case tools.ToolStatusPartial:
			return fmt.Sprintf("⚠️ 部分成功: %s", response.Text)
		default:
			return response.Text
		}
	}

	// 2) Function tool
	if fn := a.ToolRegistry.GetFunction(toolName); fn != nil {
		inputText := ""
		if v, ok := arguments["input"]; ok {
			inputText = fmt.Sprintf("%v", v)
		}

		response := a.ToolRegistry.ExecuteTool(toolName, inputText)
		_ = fn // explicit use for parity readability
		switch response.Status {
		case tools.ToolStatusError:
			code := "UNKNOWN"
			if response.ErrorInfo != nil && response.ErrorInfo["code"] != "" {
				code = response.ErrorInfo["code"]
			}
			return fmt.Sprintf("❌ 错误 [%s]: %s", code, response.Text)
		case tools.ToolStatusPartial:
			return fmt.Sprintf("⚠️ 部分成功: %s", response.Text)
		default:
			return response.Text
		}
	}

	return fmt.Sprintf("❌ 错误：未找到工具 '%s'", toolName)
}

func (a *BaseAgent) AutoSave() {
	if a.SessionStore == nil {
		return
	}

	_, _ = a.SessionStore.Save(
		a.GetAgentConfig(),
		a.HistoryManager.GetHistory(),
		a.ComputeToolSchemaHash(),
		a.GetReadCache(),
		a.SessionMetadata,
		"session-auto",
	)
}

func (a *BaseAgent) SaveSession(sessionName string) (string, error) {
	if a.SessionStore == nil {
		return "", fmt.Errorf("会话持久化未启用，请在 Config 中设置 session_enabled=true")
	}
	a.SessionMetadata["duration_seconds"] = time.Since(a.StartTime).Seconds()
	return a.SessionStore.Save(
		a.GetAgentConfig(),
		a.HistoryManager.GetHistory(),
		a.ComputeToolSchemaHash(),
		a.GetReadCache(),
		a.SessionMetadata,
		sessionName,
	)
}

func (a *BaseAgent) LoadSession(filepath string, checkConsistency bool) error {
	if a.SessionStore == nil {
		return fmt.Errorf("会话持久化未启用，请在 Config 中设置 session_enabled=true")
	}
	record, err := a.SessionStore.Load(filepath)
	if err != nil {
		return err
	}

	if checkConsistency {
		configCheck := a.SessionStore.CheckConfigConsistency(record.AgentConfig, a.GetAgentConfig())
		if consistent, _ := configCheck["consistent"].(bool); !consistent {
			warnings := make([]string, 0)
			if raw, ok := configCheck["warnings"].([]string); ok {
				warnings = raw
			} else if raw, ok := configCheck["warnings"].([]any); ok {
				for _, item := range raw {
					warnings = append(warnings, fmt.Sprintf("%v", item))
				}
			}
			if len(warnings) > 0 {
				fmt.Println("⚠️ 环境配置不一致：")
				for _, warning := range warnings {
					fmt.Printf("  - %s\n", warning)
				}
				if a.TraceLogger != nil {
					a.TraceLogger.LogEvent("session_config_warning", map[string]any{
						"warnings": warnings,
					}, nil)
				}
			}
		}

		schemaCheck := a.SessionStore.CheckToolSchemaConsistency(record.ToolSchemaHash, a.ComputeToolSchemaHash())
		if changed, _ := schemaCheck["changed"].(bool); changed {
			fmt.Println("⚠️ 工具定义已变化")
			fmt.Printf("  建议：%v\n", schemaCheck["recommendation"])
			if a.TraceLogger != nil {
				a.TraceLogger.LogEvent("session_tool_schema_warning", schemaCheck, nil)
			}
		}
	}

	a.HistoryManager.Clear()
	for _, item := range record.History {
		msg, err := MessageFromMap(item)
		if err == nil {
			a.HistoryManager.Append(msg)
		}
	}
	a.SessionMetadata = record.Metadata
	a.HistoryTokenCount = a.TokenCounter.CountMessages(a.HistoryManager.GetHistory())

	if a.ToolRegistry != nil {
		a.ToolRegistry.ClearReadCache(nil)
		for filePath, metadata := range record.ReadCache {
			a.ToolRegistry.CacheReadMetadata(filePath, metadata)
		}
	}
	fmt.Printf("✅ 会话已恢复：%s\n", record.SessionID)
	return nil
}

func (a *BaseAgent) ListSessions() ([]map[string]any, error) {
	if a.SessionStore == nil {
		return []map[string]any{}, nil
	}
	return a.SessionStore.ListSessions()
}

func (a *BaseAgent) GetAgentConfig() map[string]any {
	agentType := a.AgentType
	if strings.TrimSpace(agentType) == "" {
		agentType = "BaseAgent"
	}
	llmProvider := "unknown"
	if a.LLM != nil && strings.TrimSpace(a.LLM.Provider) != "" {
		llmProvider = a.LLM.Provider
	}
	llmModel := "unknown"
	if a.LLM != nil && strings.TrimSpace(a.LLM.Model) != "" {
		llmModel = a.LLM.Model
	}

	cfg := map[string]any{
		"name":         a.Name,
		"agent_type":   agentType,
		"llm_provider": llmProvider,
		"llm_model":    llmModel,
	}
	if a.MaxSteps > 0 {
		cfg["max_steps"] = a.MaxSteps
	}
	return cfg
}

func (a *BaseAgent) ComputeToolSchemaHash() string {
	if a.ToolRegistry == nil {
		return "no-tools"
	}

	toolSignature := map[string]any{}
	for _, toolName := range a.ToolRegistry.ListTools() {
		tool := a.ToolRegistry.GetTool(toolName)
		if tool == nil {
			continue
		}

		params := make([]string, 0, len(tool.GetParameters()))
		for _, p := range tool.GetParameters() {
			params = append(params, p.Name)
		}

		description := tool.GetDescription()
		if len(description) > 100 {
			description = description[:100]
		}
		toolSignature[toolName] = map[string]any{
			"name":        tool.GetName(),
			"description": description,
			"parameters":  params,
		}
	}

	payload, _ := json.Marshal(toolSignature)
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:])[:16]
}

func (a *BaseAgent) GetReadCache() map[string]map[string]any {
	if a.ToolRegistry == nil {
		return map[string]map[string]any{}
	}
	cache := a.ToolRegistry.ReadMetadataCache()
	if cache == nil {
		return map[string]map[string]any{}
	}
	return cache
}

func (a *BaseAgent) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	originalHistory := a.GetHistory()
	a.ClearHistory()

	disabledTools, disabledFunctions := a.applyToolFilter(toolFilter)

	originalMaxSteps := a.MaxSteps
	if maxStepsOverride != nil && *maxStepsOverride > 0 && a.MaxSteps > 0 {
		a.MaxSteps = *maxStepsOverride
	}

	start := time.Now()
	result, err := a.Run(task, nil)
	duration := time.Since(start).Seconds()

	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
		result = "执行失败: " + errorMsg
	}

	isolatedHistory := a.GetHistory()
	metadata := a.getSubagentMetadata(isolatedHistory, duration, errorMsg)

	a.HistoryManager.Clear()
	for _, msg := range originalHistory {
		a.HistoryManager.Append(msg)
	}
	a.HistoryTokenCount = a.TokenCounter.CountMessages(a.HistoryManager.GetHistory())

	a.restoreTools(disabledTools, disabledFunctions)

	if originalMaxSteps > 0 {
		a.MaxSteps = originalMaxSteps
	}

	if returnSummary {
		summary := a.GenerateSubagentSummary(task, result, metadata)
		return map[string]any{"success": success, "summary": summary, "metadata": metadata}
	}
	return map[string]any{"success": success, "result": result, "metadata": metadata}
}

func (a *BaseAgent) applyToolFilter(toolFilter tools.ToolFilter) (map[string]tools.Tool, map[string]tools.FunctionTool) {
	disabledTools := map[string]tools.Tool{}
	disabledFunctions := map[string]tools.FunctionTool{}
	if toolFilter == nil || a.ToolRegistry == nil {
		return disabledTools, disabledFunctions
	}

	originalTools := a.ToolRegistry.ListTools()
	filtered := toolFilter.Filter(originalTools)
	allowed := map[string]struct{}{}
	for _, name := range filtered {
		allowed[name] = struct{}{}
	}

	allFunctions := a.ToolRegistry.GetAllFunctions()
	for _, name := range originalTools {
		if _, ok := allowed[name]; ok {
			continue
		}
		if tool := a.ToolRegistry.GetTool(name); tool != nil {
			disabledTools[name] = tool
		}
		if fn, ok := allFunctions[name]; ok {
			disabledFunctions[name] = fn
		}
		a.ToolRegistry.UnregisterTool(name)
	}
	return disabledTools, disabledFunctions
}

func (a *BaseAgent) restoreTools(disabledTools map[string]tools.Tool, disabledFunctions map[string]tools.FunctionTool) {
	if a.ToolRegistry == nil {
		return
	}
	for _, tool := range disabledTools {
		a.ToolRegistry.RegisterTool(tool, false)
	}
	for name, fn := range disabledFunctions {
		a.ToolRegistry.RegisterFunction(name, fn.Handler, fn.Description)
	}
}

func (a *BaseAgent) getSubagentMetadata(history []Message, duration float64, errorMsg string) map[string]any {
	steps := 0
	totalChars := 0
	for _, msg := range history {
		if msg.Role == MessageRoleAssistant {
			steps++
		}
		totalChars += len(msg.Content)
	}

	metadata := map[string]any{
		"steps":            steps,
		"tokens":           totalChars / 4,
		"duration_seconds": float64(int(duration*100)) / 100,
		"tools_used":       extractToolsFromHistory(history),
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	return metadata
}

func extractToolsFromHistory(history []Message) []string {
	toolSet := map[string]struct{}{}
	re := regexp.MustCompile(`Action:\s*(\w+)\[`)

	for _, msg := range history {
		if msg.Metadata != nil {
			if rawCalls, ok := msg.Metadata["tool_calls"].([]any); ok {
				for _, raw := range rawCalls {
					callMap, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					if function, ok := callMap["function"].(map[string]any); ok {
						if name, ok := function["name"].(string); ok && name != "" {
							toolSet[name] = struct{}{}
						}
					}
				}
			}
		}

		if msg.Role == MessageRoleAssistant {
			matches := re.FindAllStringSubmatch(msg.Content, -1)
			for _, m := range matches {
				if len(m) > 1 && m[1] != "" {
					toolSet[m[1]] = struct{}{}
				}
			}
		}
	}

	out := make([]string, 0, len(toolSet))
	for name := range toolSet {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (a *BaseAgent) GenerateSubagentSummary(task, result string, metadata map[string]any) string {
	preview := result
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	parts := []string{
		fmt.Sprintf("任务: %s", task),
		fmt.Sprintf("结果: %s", preview),
		fmt.Sprintf("步数: %v", metadata["steps"]),
		fmt.Sprintf("耗时: %v秒", metadata["duration_seconds"]),
	}
	if toolsUsed, ok := metadata["tools_used"].([]string); ok && len(toolsUsed) > 0 {
		parts = append(parts, fmt.Sprintf("工具: %s", strings.Join(toolsUsed, ", ")))
	}
	if errMsg, ok := metadata["error"].(string); ok && errMsg != "" {
		parts = append(parts, fmt.Sprintf("错误: %s", errMsg))
	}
	return strings.Join(parts, "\n")
}

func (a *BaseAgent) String() string {
	provider := ""
	if a.LLM != nil {
		provider = a.LLM.Provider
	}
	return fmt.Sprintf("Agent(name=%s, provider=%s)", a.Name, provider)
}
