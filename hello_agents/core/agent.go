package core

import (
	stdctx "context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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
	ArunStream(inputText string, kwargs map[string]any, hooks ...Hooks) <-chan AgentEvent
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
	runDelegate  func(inputText string, kwargs map[string]any) (string, error)
	getMaxSteps  func() int
	setMaxSteps  func(v int)
	hasMaxSteps  bool

	HistoryTokenCount int
	SessionMetadata   map[string]any
	StartTime         time.Time
}

type disabledToolEntry struct {
	Tool tools.Tool
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
				map[string]any{"compressed_at": nowPythonISOTime()},
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
			"created_at":       nowPythonISOTime(),
			"total_tokens":     0,
			"total_steps":      0,
			"duration_seconds": 0,
		},
	}
	agent.getMaxSteps = func() int {
		return agent.MaxSteps
	}
	agent.setMaxSteps = func(v int) {
		agent.MaxSteps = v
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

func (a *BaseAgent) SetRunDelegate(runDelegate func(inputText string, kwargs map[string]any) (string, error)) {
	a.runDelegate = runDelegate
}

func (a *BaseAgent) SetMaxStepAccessors(getter func() int, setter func(v int)) {
	a.getMaxSteps = getter
	a.setMaxSteps = setter
	a.hasMaxSteps = true
}

func (a *BaseAgent) Arun(inputText string, hooks Hooks, kwargs map[string]any) (string, error) {
	if err := a.emitEvent(AgentStart, hooks.OnStart, map[string]any{"input_text": inputText}); err != nil {
		return "", err
	}

	runner := a.Run
	if a.runDelegate != nil {
		runner = a.runDelegate
	}
	result, err := runner(inputText, kwargs)
	if err != nil {
		_ = a.emitEvent(AgentError, hooks.OnError, map[string]any{"error": err.Error(), "error_type": "AgentError"})
		return "", err
	}

	_ = a.emitEvent(AgentFinish, hooks.OnFinish, map[string]any{"result": result})
	return result, nil
}

func (a *BaseAgent) ArunStream(inputText string, kwargs map[string]any, hooks ...Hooks) <-chan AgentEvent {
	activeHooks := Hooks{}
	if len(hooks) > 0 {
		activeHooks = hooks[0]
	}

	out := make(chan AgentEvent, 2)
	go func() {
		defer close(out)
		startEvent := NewAgentEvent(AgentStart, a.Name, map[string]any{"input_text": inputText})
		if activeHooks.OnStart != nil {
			_ = activeHooks.OnStart(startEvent)
		}
		out <- startEvent
		runner := a.Run
		if a.runDelegate != nil {
			runner = a.runDelegate
		}
		result, err := runner(inputText, kwargs)
		if err != nil {
			errorEvent := NewAgentEvent(AgentError, a.Name, map[string]any{"error": err.Error(), "error_type": "AgentError"})
			if activeHooks.OnError != nil {
				_ = activeHooks.OnError(errorEvent)
			}
			out <- errorEvent
			return
		}
		finishEvent := NewAgentEvent(AgentFinish, a.Name, map[string]any{"result": result})
		if activeHooks.OnFinish != nil {
			_ = activeHooks.OnFinish(finishEvent)
		}
		out <- finishEvent
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
		if historyLen%a.Config.AutoSaveInterval == 0 {
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
		fmt.Printf("⚠️ 智能摘要生成失败: %v，使用简单摘要\n", err)
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
		fmt.Printf("⚠️ 智能摘要生成失败: %v，使用简单摘要\n", err)
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

	model := a.Config.SummaryLLMModel

	maxTokens := a.Config.SummaryMaxTokens

	llm, err := NewHelloAgentsLLM(
		model,
		"",
		"",
		a.Config.SummaryTemperature,
		&maxTokens,
		nil,
		map[string]any{"provider": a.Config.SummaryLLMProvider},
	)
	if err != nil {
		return nil, err
	}

	a.summaryLLM = llm
	return a.summaryLLM, nil
}

func (a *BaseAgent) mapParameterType(paramType string) string {
	normalized := strings.ToLower(paramType)
	switch normalized {
	case "string", "number", "integer", "boolean", "array", "object":
		return normalized
	default:
		return "string"
	}
}

func (a *BaseAgent) convertParameterTypes(toolName string, input map[string]any) map[string]any {
	if a.ToolRegistry == nil {
		return input
	}

	tool := a.ToolRegistry.GetTool(toolName)
	if tool == nil {
		return input
	}

	params := tool.GetParameters()
	typeMap := map[string]string{}
	for _, p := range params {
		typeMap[p.Name] = strings.ToLower(p.Type)
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
			case bool:
				if v {
					converted[key] = float64(1)
				} else {
					converted[key] = float64(0)
				}
			case string:
				parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
				if err == nil {
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
			case bool:
				if v {
					converted[key] = 1
				} else {
					converted[key] = 0
				}
			case string:
				parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
				if err == nil {
					converted[key] = int(parsed)
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
				lower := strings.ToLower(v)
				converted[key] = lower == "true" || lower == "1" || lower == "yes"
			case int:
				converted[key] = v != 0
			default:
				converted[key] = pythonTruthy(value)
			}
		default:
			converted[key] = value
		}
	}
	return converted
}

func pythonTruthy(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return len(v) > 0
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case uintptr:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return rv.Len() > 0
	case reflect.Chan, reflect.Func, reflect.Pointer, reflect.Interface:
		return !rv.IsNil()
	default:
		return true
	}
}

func (a *BaseAgent) BuildToolSchemas() []map[string]any {
	if a.ToolRegistry == nil {
		return []map[string]any{}
	}

	schemas := make([]map[string]any, 0)

	// Tool object schemas.
	for _, tool := range a.ToolRegistry.GetAllTools() {
		params := []tools.ToolParameter{}
		func() {
			defer func() {
				if recover() != nil {
					params = []tools.ToolParameter{}
				}
			}()
			params = tool.GetParameters()
		}()

		properties := map[string]any{}
		required := make([]string, 0)

		for _, param := range params {
			prop := map[string]any{
				"type":        a.mapParameterType(param.Type),
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

	// Function tools (keep registration order, aligned with Python dict order).
	functionMap := a.ToolRegistry.GetAllFunctions()
	for _, name := range a.ToolRegistry.ListFunctions() {
		info, ok := functionMap[name]
		if !ok {
			continue
		}
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
		var response tools.ToolResponse
		var recovered any
		func() {
			defer func() {
				if p := recover(); p != nil {
					recovered = p
				}
			}()
			typedArguments := a.convertParameterTypes(toolName, arguments)
			response = tool.RunWithTiming(typedArguments)
		}()
		if recovered != nil {
			return fmt.Sprintf("❌ 工具调用失败：%v", recovered)
		}

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

		var response tools.ToolResponse
		var recovered any
		func() {
			defer func() {
				if p := recover(); p != nil {
					recovered = p
				}
			}()
			response = a.ToolRegistry.ExecuteTool(toolName, inputText)
		}()
		if recovered != nil {
			return fmt.Sprintf("❌ 工具调用失败：%v", recovered)
		}

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

	_, err := a.SessionStore.Save(
		a.GetAgentConfig(),
		a.HistoryManager.GetHistory(),
		a.ComputeToolSchemaHash(),
		a.GetReadCache(),
		a.SessionMetadata,
		"session-auto",
	)
	if err != nil && a.Config.Debug {
		fmt.Printf("⚠️ 自动保存失败: %v\n", err)
	}
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

	if a.ToolRegistry != nil && len(record.ReadCache) > 0 {
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
	if agentType == "" {
		agentType = "BaseAgent"
	}
	llmProvider := "unknown"
	if a.LLM != nil && a.LLM.Provider != "" {
		llmProvider = a.LLM.Provider
	}
	llmModel := "unknown"
	if a.LLM != nil && a.LLM.Model != "" {
		llmModel = a.LLM.Model
	}

	cfg := map[string]any{
		"name":         a.Name,
		"agent_type":   agentType,
		"llm_provider": llmProvider,
		"llm_model":    llmModel,
	}
	if a.hasMaxSteps {
		if a.getMaxSteps != nil {
			cfg["max_steps"] = a.getMaxSteps()
		} else {
			cfg["max_steps"] = a.MaxSteps
		}
	}
	return cfg
}

func (a *BaseAgent) ComputeToolSchemaHash() string {
	if a.ToolRegistry == nil {
		return "no-tools"
	}

	toolSignature := map[string]any{}
	toolNames := append([]string(nil), a.ToolRegistry.ListTools()...)
	sort.Strings(toolNames)
	for _, toolName := range toolNames {
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

	disabledTools := a.applyToolFilter(toolFilter)

	var originalMaxSteps *int
	applyMaxSteps := func(v int) {
		if a.setMaxSteps != nil {
			a.setMaxSteps(v)
			return
		}
		a.MaxSteps = v
	}
	currentMaxSteps := func() int {
		if a.getMaxSteps != nil {
			return a.getMaxSteps()
		}
		return a.MaxSteps
	}
	if maxStepsOverride != nil && a.hasMaxSteps {
		current := currentMaxSteps()
		originalMaxSteps = &current
		applyMaxSteps(*maxStepsOverride)
	}

	start := time.Now()
	runner := a.Run
	if a.runDelegate != nil {
		runner = a.runDelegate
	}
	result, err := runner(task, nil)
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

	a.restoreTools(disabledTools)

	if originalMaxSteps != nil {
		applyMaxSteps(*originalMaxSteps)
	}

	if returnSummary {
		summary := a.GenerateSubagentSummary(task, result, metadata)
		return map[string]any{"success": success, "summary": summary, "metadata": metadata}
	}
	return map[string]any{"success": success, "result": result, "metadata": metadata}
}

func (a *BaseAgent) applyToolFilter(toolFilter tools.ToolFilter) []disabledToolEntry {
	disabledTools := make([]disabledToolEntry, 0)
	if toolFilter == nil || a.ToolRegistry == nil {
		return disabledTools
	}

	originalTools := a.ToolRegistry.ListTools()
	filtered := toolFilter.Filter(originalTools)
	allowed := map[string]struct{}{}
	for _, name := range filtered {
		allowed[name] = struct{}{}
	}

	for _, name := range originalTools {
		if _, ok := allowed[name]; ok {
			continue
		}
		if tool := a.ToolRegistry.GetTool(name); tool != nil {
			if a.ToolRegistry.DisableTool(name) {
				disabledTools = append(disabledTools, disabledToolEntry{
					Tool: tool,
				})
			}
		}
	}
	return disabledTools
}

func (a *BaseAgent) restoreTools(disabledTools []disabledToolEntry) {
	if a.ToolRegistry == nil {
		return
	}
	for _, entry := range disabledTools {
		if entry.Tool == nil {
			continue
		}
		a.ToolRegistry.RegisterTool(entry.Tool, false)
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
		"duration_seconds": math.Round(duration*100) / 100,
		"tools_used":       a.extractToolsFromHistory(history),
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	return metadata
}

func (a *BaseAgent) extractToolsFromHistory(history []Message) []string {
	toolSet := map[string]struct{}{}
	re := regexp.MustCompile(`Action:\s*(\w+)\[`)

	for _, msg := range history {
		if msg.Metadata != nil {
			if rawCalls, ok := msg.Metadata["tool_calls"]; ok {
				extractToolCallName := func(call map[string]any) {
					function, ok := call["function"].(map[string]any)
					if !ok {
						return
					}
					name, ok := function["name"].(string)
					if ok {
						toolSet[name] = struct{}{}
					}
				}
				switch calls := rawCalls.(type) {
				case []any:
					for _, raw := range calls {
						callMap, ok := raw.(map[string]any)
						if !ok {
							continue
						}
						extractToolCallName(callMap)
					}
				case []map[string]any:
					for _, callMap := range calls {
						extractToolCallName(callMap)
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

// RegisterTaskTool mirrors Python _register_task_tool flow.
func (a *BaseAgent) RegisterTaskTool(agentFactory builtin.AgentFactory) {
	if a.ToolRegistry == nil || agentFactory == nil {
		return
	}
	taskTool := builtin.NewTaskTool(agentFactory, a.ToolRegistry)
	a.ToolRegistry.RegisterTool(taskTool, false)
}

// RegisterTodoWriteTool mirrors Python _register_todowrite_tool.
func (a *BaseAgent) RegisterTodoWriteTool() {
	if a.ToolRegistry == nil {
		return
	}
	todoTool := builtin.NewTodoWriteTool(".", a.Config.TodoWritePersistenceDir)
	a.ToolRegistry.RegisterTool(todoTool, false)
}

// RegisterDevLogTool mirrors Python _register_devlog_tool.
func (a *BaseAgent) RegisterDevLogTool() {
	if a.ToolRegistry == nil {
		return
	}
	sessionID := ""
	if a.TraceLogger != nil {
		sessionID = a.TraceLogger.SessionID
	}
	if sessionID == "" {
		sessionID = a.GenerateSessionID()
	}
	devlogTool := builtin.NewDevLogTool(
		sessionID,
		a.Name,
		".",
		a.Config.DevLogPersistenceDir,
	)
	a.ToolRegistry.RegisterTool(devlogTool, false)
}

// GenerateSessionID mirrors Python _generate_session_id fallback logic.
func (a *BaseAgent) GenerateSessionID() string {
	now := time.Now().Format("20060102-150405")
	randomBytes := make([]byte, 2)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Sprintf("s-%s-%04x", now, time.Now().UnixNano()&0xffff)
	}
	return fmt.Sprintf("s-%s-%s", now, hex.EncodeToString(randomBytes))
}

// CreateLightLLM mirrors Python _create_light_llm behavior.
func (a *BaseAgent) CreateLightLLM() *HelloAgentsLLM {
	if a.LLM == nil {
		return nil
	}

	model := a.Config.SubagentLightLLMModel
	lightLLM, err := NewHelloAgentsLLM(
		model,
		"",
		"",
		a.LLM.Temperature,
		a.LLM.MaxTokens,
		nil,
		map[string]any{"provider": a.Config.SubagentLightLLMProvider},
	)
	if err != nil {
		return nil
	}
	return lightLLM
}

func (a *BaseAgent) String() string {
	model := ""
	if a.LLM != nil {
		model = a.LLM.Model
	}
	return fmt.Sprintf("Agent(name=%s, model=%s)", a.Name, model)
}

func (a *BaseAgent) Repr() string {
	return a.String()
}
