package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

type typedParamTool struct {
	tools.BaseTool
}

type panicParametersTool struct {
	tools.BaseTool
}

type panicExecuteTool struct {
	tools.BaseTool
}

type noopTool struct {
	tools.BaseTool
}

func newTypedParamTool() *typedParamTool {
	base := tools.NewBaseTool("typed_tool", "typed conversion test tool", false)
	base.Parameters = map[string]tools.ToolParameter{
		"num": {
			Name:        "num",
			Type:        "number",
			Description: "number parameter",
			Required:    false,
		},
		"count": {
			Name:        "count",
			Type:        "integer",
			Description: "integer parameter",
			Required:    false,
		},
		"flag": {
			Name:        "flag",
			Type:        "boolean",
			Description: "boolean parameter",
			Required:    false,
		},
	}
	return &typedParamTool{BaseTool: base}
}

func newPanicParametersTool() *panicParametersTool {
	base := tools.NewBaseTool("panic_params", "panic params tool", false)
	return &panicParametersTool{BaseTool: base}
}

func (t *panicParametersTool) GetParameters() []tools.ToolParameter {
	panic("boom")
}

func newPanicExecuteTool() *panicExecuteTool {
	base := tools.NewBaseTool("panic_execute", "panic execute tool", false)
	return &panicExecuteTool{BaseTool: base}
}

func newNoopTool(name string) *noopTool {
	base := tools.NewBaseTool(name, "noop", false)
	return &noopTool{BaseTool: base}
}

func (t *panicExecuteTool) RunWithTiming(parameters map[string]any) tools.ToolResponse {
	panic("boom")
}

func (t *noopTool) Run(parameters map[string]any) tools.ToolResponse {
	return tools.Success("ok", nil, nil, nil)
}

func TestBaseAgentArunUsesRunDelegate(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	called := false
	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		called = true
		return "delegated:" + inputText, nil
	})

	result, err := agent.Arun("hello", core.Hooks{}, nil)
	if err != nil {
		t.Fatalf("Arun() error = %v", err)
	}
	if !called {
		t.Fatalf("Arun() did not invoke run delegate")
	}
	if result != "delegated:hello" {
		t.Fatalf("Arun() result = %q, want %q", result, "delegated:hello")
	}
}

func TestBaseAgentArunStreamUsesRunDelegate(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		return "stream:" + inputText, nil
	})

	events := agent.ArunStream("hello", nil)
	finishSeen := false
	for event := range events {
		if event.Type != core.AgentFinish {
			continue
		}
		finishSeen = true
		if got := event.Data["result"]; got != "stream:hello" {
			t.Fatalf("AgentFinish result = %v, want %q", got, "stream:hello")
		}
	}
	if !finishSeen {
		t.Fatalf("AgentFinish event not emitted")
	}
}

func TestBaseAgentRunAsSubagentMaxStepsOverrideAndRestore(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	maxSteps := 5
	agent.SetMaxStepAccessors(
		func() int { return maxSteps },
		func(v int) { maxSteps = v },
	)
	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		if maxSteps != 0 {
			t.Fatalf("run() saw maxSteps = %d, want 0 during subagent run", maxSteps)
		}
		return "ok", nil
	})

	zero := 0
	result := agent.RunAsSubagent("task", nil, true, &zero)
	if success, _ := result["success"].(bool); !success {
		t.Fatalf("RunAsSubagent() success = %v, want true", result["success"])
	}
	if maxSteps != 5 {
		t.Fatalf("maxSteps after restore = %d, want 5", maxSteps)
	}
}

func TestBaseAgentGetAgentConfigIncludesMaxStepsOnlyWhenEnabled(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	agent.MaxSteps = 9
	configWithoutMax := agent.GetAgentConfig()
	if _, exists := configWithoutMax["max_steps"]; exists {
		t.Fatalf("GetAgentConfig() should not include max_steps when accessor not enabled")
	}

	maxSteps := 0
	agent.SetMaxStepAccessors(
		func() int { return maxSteps },
		func(v int) { maxSteps = v },
	)
	configWithMax := agent.GetAgentConfig()
	got, exists := configWithMax["max_steps"]
	if !exists {
		t.Fatalf("GetAgentConfig() missing max_steps when accessor enabled")
	}
	if got != 0 {
		t.Fatalf("GetAgentConfig() max_steps = %v, want 0", got)
	}
}

func TestBaseAgentConvertParameterTypesMatchesPythonSemantics(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(newTypedParamTool(), false)

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, registry)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	converted := agent.ExportConvertParameterTypes("typed_tool", map[string]any{
		"num":   "12.3abc",
		"count": "7.9",
		"flag":  []any{},
	})

	if got := converted["num"]; got != "12.3abc" {
		t.Fatalf("number parse should fail strictly, got %v", got)
	}
	if got := converted["count"]; got != "7.9" {
		t.Fatalf("integer parse should fail strictly, got %v", got)
	}
	if got, ok := converted["flag"].(bool); !ok || got {
		t.Fatalf("empty slice should convert to false, got %#v", converted["flag"])
	}

	convertedTruthy := agent.ExportConvertParameterTypes("typed_tool", map[string]any{
		"flag": map[string]any{"k": 1},
	})
	if got, ok := convertedTruthy["flag"].(bool); !ok || !got {
		t.Fatalf("non-empty map should convert to true, got %#v", convertedTruthy["flag"])
	}

	convertedStringBool := agent.ExportConvertParameterTypes("typed_tool", map[string]any{
		"flag": " true ",
	})
	if got, ok := convertedStringBool["flag"].(bool); !ok || got {
		t.Fatalf("string with surrounding spaces should be false like python lower() check, got %#v", convertedStringBool["flag"])
	}

	if got := agent.ExportMapParameterType(" number "); got != "string" {
		t.Fatalf("mapParameterType should not trim whitespace, got %q", got)
	}
}

func TestBaseAgentBuildToolSchemasHandlesToolParameterPanics(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(newPanicParametersTool(), false)

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, registry)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	schemas := agent.BuildToolSchemas()
	if len(schemas) != 1 {
		t.Fatalf("BuildToolSchemas() len = %d, want 1", len(schemas))
	}
	function, _ := schemas[0]["function"].(map[string]any)
	params, _ := function["parameters"].(map[string]any)
	properties, _ := params["properties"].(map[string]any)
	if len(properties) != 0 {
		t.Fatalf("properties len = %d, want 0 when GetParameters panics", len(properties))
	}
}

func TestBaseAgentExecuteToolCallHandlesToolPanics(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(newPanicExecuteTool(), false)

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, registry)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	result := agent.ExecuteToolCall("panic_execute", map[string]any{})
	if result != "❌ 工具调用失败：boom" {
		t.Fatalf("ExecuteToolCall() = %q, want panic fallback", result)
	}
}

func TestBaseAgentCreateLightLLMUsesEnvLikePython(t *testing.T) {
	t.Setenv("LLM_API_KEY", "env-key")
	t.Setenv("LLM_BASE_URL", "https://env-base.example/v1")

	llm := core.NewLLMFromAdapter("main-model", "main-key", "https://main-base.example/v1", 30, 0.2, nil)
	llm.Provider = "mock"

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	cfg.SubagentLightLLMProvider = "deepseek"
	cfg.SubagentLightLLMModel = "light-model"

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	light := agent.CreateLightLLM()
	if light == nil {
		t.Fatalf("CreateLightLLM() returned nil")
	}
	if light.APIKey != "env-key" {
		t.Fatalf("light APIKey = %q, want env key", light.APIKey)
	}
	if light.BaseURL != "https://env-base.example/v1" {
		t.Fatalf("light BaseURL = %q, want env base", light.BaseURL)
	}
	if light.Model != "light-model" {
		t.Fatalf("light model = %q, want %q", light.Model, "light-model")
	}
}

func TestBaseAgentStringUsesModelLikePython(t *testing.T) {
	agent := &core.BaseAgent{
		Name: "tester",
		LLM: core.NewLLMFromAdapter("gpt-test", "", "", 0, 0, nil),
	}
	agent.LLM.Provider = "mock"
	if got := agent.String(); got != "Agent(name=tester, model=gpt-test)" {
		t.Fatalf("String() = %q, want %q", got, "Agent(name=tester, model=gpt-test)")
	}
}

func TestBaseAgentGetSummaryLLMUsesSummaryConfigAndEnv(t *testing.T) {
	t.Setenv("LLM_API_KEY", "env-key")
	t.Setenv("LLM_BASE_URL", "https://env-base.example/v1")

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false
	cfg.SummaryLLMProvider = "deepseek"
	cfg.SummaryLLMModel = "deepseek-chat"

	mainLLM := core.NewLLMFromAdapter("main-model", "main-key", "https://main-base.example/v1", 0, 0.9, nil)
	mainLLM.Provider = "openai"

	agent, err := core.NewBaseAgent("tester", mainLLM, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	summaryLLM, err := agent.ExportGetSummaryLLM()
	if err != nil {
		t.Fatalf("getSummaryLLM() error = %v", err)
	}
	if summaryLLM.Model != "deepseek-chat" {
		t.Fatalf("summary model = %q, want %q", summaryLLM.Model, "deepseek-chat")
	}
	if summaryLLM.APIKey != "env-key" {
		t.Fatalf("summary APIKey = %q, want env key", summaryLLM.APIKey)
	}
	if summaryLLM.BaseURL != "https://env-base.example/v1" {
		t.Fatalf("summary BaseURL = %q, want env base", summaryLLM.BaseURL)
	}
}

func TestExtractToolsFromHistoryKeepsEmptyFunctionNameLikePythonSet(t *testing.T) {
	agent := &core.BaseAgent{}
	history := []core.Message{
		{
			Role: core.MessageRoleAssistant,
			Metadata: map[string]any{
				"tool_calls": []map[string]any{
					{"function": map[string]any{"name": ""}},
				},
			},
		},
	}

	toolsUsed := agent.ExportExtractToolsFromHistory(history)
	if len(toolsUsed) != 1 || toolsUsed[0] != "" {
		t.Fatalf("extractToolsFromHistory() = %#v, want [\"\"]", toolsUsed)
	}
}

func TestAddMessagePanicsWhenAutoSaveIntervalIsZeroLikePythonModuloError(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = true
	cfg.SessionDir = t.TempDir()
	cfg.AutoSaveEnabled = true
	cfg.AutoSaveInterval = 0

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatalf("AddMessage() should panic when auto_save_interval is 0")
		}
	}()
	agent.AddMessage(core.NewMessage("hello", core.MessageRoleUser, nil))
}

func TestLoadSessionKeepsExistingReadCacheWhenSavedReadCacheIsEmpty(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = true
	cfg.SessionDir = t.TempDir()

	registry := tools.NewToolRegistry(nil)
	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, registry)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	registry.CacheReadMetadata("a.txt", map[string]any{"file_size_bytes": 1})

	path, err := agent.SessionStore.Save(
		agent.GetAgentConfig(),
		nil,
		"hash",
		map[string]map[string]any{},
		map[string]any{},
		"empty-cache",
	)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := agent.LoadSession(path, false); err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if got := registry.GetReadMetadata("a.txt"); got == nil {
		t.Fatalf("existing read cache should be preserved when saved read_cache is empty")
	}
}

func TestRunAsSubagentKeepsMainHistoryIsolated(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, nil)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}

	agent.AddMessage(core.NewMessage("main-user", core.MessageRoleUser, nil))
	agent.AddMessage(core.NewMessage("main-assistant", core.MessageRoleAssistant, nil))
	original := agent.GetHistory()

	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		agent.AddMessage(core.NewMessage(inputText, core.MessageRoleUser, nil))
		agent.AddMessage(core.NewMessage("subagent-result", core.MessageRoleAssistant, nil))
		return "subagent-result", nil
	})

	result := agent.RunAsSubagent("sub-task", nil, true, nil)
	if success, _ := result["success"].(bool); !success {
		t.Fatalf("RunAsSubagent() success = %v, want true", result["success"])
	}

	finalHistory := agent.GetHistory()
	if len(finalHistory) != len(original) {
		t.Fatalf("history len after subagent = %d, want %d", len(finalHistory), len(original))
	}
	if finalHistory[0].Content != "main-user" || finalHistory[1].Content != "main-assistant" {
		t.Fatalf("main history should be restored, got %#v", finalHistory)
	}
}

func TestRunAsSubagentRestoresToolsAfterFilter(t *testing.T) {
	llm := core.NewLLMFromAdapter("mock-model", "", "", 0, 0, nil)
	llm.Provider = "mock"
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SessionEnabled = false
	cfg.SkillsEnabled = false

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(newNoopTool("Read"), false)
	registry.RegisterTool(newNoopTool("Write"), false)
	registry.RegisterTool(newNoopTool("Bash"), false)

	agent, err := core.NewBaseAgent("tester", llm, "", &cfg, registry)
	if err != nil {
		t.Fatalf("NewBaseAgent() error = %v", err)
	}
	agent.SetRunDelegate(func(inputText string, kwargs map[string]any) (string, error) {
		return "ok", nil
	})

	filter := tools.NewReadOnlyFilter(nil)
	_ = agent.RunAsSubagent("task", filter, true, nil)

	names := registry.ListTools()
	want := map[string]bool{"Read": true, "Write": true, "Bash": true}
	for _, name := range names {
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("tools should be fully restored, missing: %#v", want)
	}
}

func TestNewBaseAgentReturnsErrorWhenTraceLoggerInitFailsLikePython(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = true
	cfg.TraceDir = filepath.Join(blocker, "traces")
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	_, err := core.NewBaseAgent("tester", core.NewLLMFromAdapter("m", "", "", 0, 0, nil), "", &cfg, nil)
	if err == nil {
		t.Fatalf("NewBaseAgent() should return error when trace logger init fails")
	}
}
