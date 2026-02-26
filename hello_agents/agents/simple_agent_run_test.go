package agents

import (
	"strings"
	"testing"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/core/testutil"
	"helloagents-go/hello_agents/tools"
)

func noTraceConfig() *core.Config {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	return &cfg
}

func TestSimpleAgentRunWithoutTools(t *testing.T) {
	llm := testutil.NewMockLLM("你好！我是AI助手。")
	agent, err := NewSimpleAgent("test-agent", llm, "你是一个AI助手", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("NewSimpleAgent error: %v", err)
	}

	result, err := agent.Run("你好", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result != "你好！我是AI助手。" {
		t.Fatalf("result = %q, want mock response", result)
	}

	history := agent.GetHistory()
	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2 (user + assistant)", len(history))
	}
	if history[0].Role != core.MessageRoleUser {
		t.Fatalf("history[0].Role = %q, want user", history[0].Role)
	}
	if history[1].Role != core.MessageRoleAssistant {
		t.Fatalf("history[1].Role = %q, want assistant", history[1].Role)
	}
}

func TestSimpleAgentRunWithToolCalls(t *testing.T) {
	toolCallResp := testutil.MockToolCallResponse("calculator", "call-1", map[string]any{
		"expression": "2+2",
	})
	finalResp := testutil.MockTextResponse("计算结果是 4。")

	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{toolCallResp, finalResp},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	registry := tools.NewToolRegistry(nil)
	calcTool := &mockCalculatorTool{}
	registry.RegisterTool(calcTool, false)

	agent, err := NewSimpleAgentWithOptions("test-agent", llm, "你是一个AI助手", noTraceConfig(), registry, true, 3)
	if err != nil {
		t.Fatalf("NewSimpleAgent error: %v", err)
	}

	result, err := agent.Run("2+2等于几？", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result != "计算结果是 4。" {
		t.Fatalf("result = %q, want final response", result)
	}

	if adapter.InvokedCount < 2 {
		t.Fatalf("InvokedCount = %d, want at least 2 (tool call + final)", adapter.InvokedCount)
	}
}

func TestSimpleAgentRunPreservesHistory(t *testing.T) {
	llm := testutil.NewMockLLM("回答1", "回答2")
	agent, err := NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	_, _ = agent.Run("问题1", nil)
	_, _ = agent.Run("问题2", nil)

	history := agent.GetHistory()
	if len(history) != 4 {
		t.Fatalf("history length = %d, want 4", len(history))
	}
}

func TestSimpleAgentClearHistory(t *testing.T) {
	llm := testutil.NewMockLLM("hello")
	agent, err := NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	_, _ = agent.Run("hi", nil)
	agent.ClearHistory()

	if len(agent.GetHistory()) != 0 {
		t.Fatalf("history should be empty after ClearHistory()")
	}
}

func TestSimpleAgentStreamRun(t *testing.T) {
	adapter := &testutil.MockLLMAdapter{
		StreamChunks: []string{"Hello", " ", "World"},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	agent, err := NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	chunkCh, errCh := agent.StreamRun("hi", nil)
	var chunks []string
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamRun error: %v", err)
		}
	}

	full := strings.Join(chunks, "")
	if full != "Hello World" {
		t.Fatalf("streamed result = %q, want 'Hello World'", full)
	}
}

func TestSimpleAgentArun(t *testing.T) {
	llm := testutil.NewMockLLM("arun-result")
	agent, err := NewSimpleAgent("test", llm, "system", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var startCalled, finishCalled bool
	hooks := core.Hooks{
		OnStart:  func(e core.AgentEvent) error { startCalled = true; return nil },
		OnFinish: func(e core.AgentEvent) error { finishCalled = true; return nil },
	}

	result, err := agent.Arun("hi", hooks, nil)
	if err != nil {
		t.Fatalf("Arun error: %v", err)
	}
	if result != "arun-result" {
		t.Fatalf("result = %q, want 'arun-result'", result)
	}
	if !startCalled {
		t.Fatal("OnStart hook was not called")
	}
	if !finishCalled {
		t.Fatal("OnFinish hook was not called")
	}
}

func TestSimpleAgentArunStream(t *testing.T) {
	llm := testutil.NewMockLLM("stream-result")
	agent, err := NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	events := agent.ArunStream("hi", nil)
	var types []core.EventType
	for event := range events {
		types = append(types, event.Type)
	}

	if len(types) < 2 {
		t.Fatalf("expected at least 2 events (start + finish), got %d", len(types))
	}
	if types[0] != core.AgentStart {
		t.Fatalf("first event = %q, want agent_start", types[0])
	}
}

// mockCalculatorTool is a simple tool for testing.
type mockCalculatorTool struct {
	tools.BaseTool
}

func init() {
	// Override base to prevent nil parameters
}

func newMockCalculatorTool() *mockCalculatorTool {
	t := &mockCalculatorTool{}
	t.Name = "calculator"
	t.Description = "计算数学表达式"
	t.Parameters = map[string]tools.ToolParameter{
		"expression": {
			Name:        "expression",
			Type:        "string",
			Description: "数学表达式",
			Required:    true,
		},
	}
	return t
}

func (t *mockCalculatorTool) Run(parameters map[string]any) tools.ToolResponse {
	return tools.Success("4", map[string]any{"result": 4})
}

func (t *mockCalculatorTool) GetName() string        { return "calculator" }
func (t *mockCalculatorTool) GetDescription() string  { return "计算数学表达式" }
func (t *mockCalculatorTool) GetParameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "expression", Type: "string", Description: "数学表达式", Required: true},
	}
}
func (t *mockCalculatorTool) RunWithTiming(parameters map[string]any) tools.ToolResponse {
	return t.Run(parameters)
}
func (t *mockCalculatorTool) ARun(parameters map[string]any) tools.ToolResponse {
	return t.Run(parameters)
}
func (t *mockCalculatorTool) ARunWithTiming(parameters map[string]any) tools.ToolResponse {
	return t.Run(parameters)
}
