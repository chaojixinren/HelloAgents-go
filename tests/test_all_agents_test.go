package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/core/testutil"
	"helloagents-go/hello_agents/tools"
)

// ---------------------------------------------------------------------------
// helpers (from agents/simple_agent_run_test.go)
// ---------------------------------------------------------------------------

func noTraceConfig() *core.Config {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	return &cfg
}

type mockCalculatorTool struct {
	tools.BaseTool
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
func (t *mockCalculatorTool) GetDescription() string { return "计算数学表达式" }
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

// ---------------------------------------------------------------------------
// expandable tool helper (from agents/simple_agent_test.go)
// ---------------------------------------------------------------------------

type expandableTestTool struct {
	tools.BaseTool
	sub tools.Tool
}

func newExpandableTestTool() *expandableTestTool {
	base := tools.NewBaseTool("parent", "parent", true)
	subBase := tools.NewBaseTool("child", "child", false)
	return &expandableTestTool{
		BaseTool: base,
		sub:      &subBase,
	}
}

func (t *expandableTestTool) GetExpandedTools() []tools.Tool {
	return []tools.Tool{t.sub}
}

// ---------------------------------------------------------------------------
// Factory tests (from agents/factory_test.go)
// ---------------------------------------------------------------------------

func TestDefaultSubagentFactoryAssignsZeroSubagentMaxStepsLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	cfg.SubagentMaxSteps = 0

	subagent, err := agents.DefaultSubagentFactory(
		"react",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*agents.ReActAgent)
	if !ok {
		t.Fatalf("subagent type = %T, want *ReActAgent", subagent)
	}
	if reactAgent.MaxSteps != 0 {
		t.Fatalf("MaxSteps = %d, want 0", reactAgent.MaxSteps)
	}
}

func TestDefaultSubagentFactoryKeepsOriginalAgentTypeCaseInNameLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	subagent, err := agents.DefaultSubagentFactory(
		"ReAct",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		nil,
		&cfg,
	)
	if err != nil {
		t.Fatalf("DefaultSubagentFactory() error = %v", err)
	}

	reactAgent, ok := subagent.(*agents.ReActAgent)
	if !ok {
		t.Fatalf("subagent type = %T, want *ReActAgent", subagent)
	}
	if reactAgent.Name != "subagent-ReAct" {
		t.Fatalf("Name = %q, want %q", reactAgent.Name, "subagent-ReAct")
	}
}

// ---------------------------------------------------------------------------
// SimpleAgent construction tests (from agents/simple_agent_test.go)
// ---------------------------------------------------------------------------

func TestSimpleAgentBuildMessagesKeepsWhitespaceSystemPrompt(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"   ",
		&cfg,
		nil,
		false,
		3,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}

	messages := agent.ExportBuildMessages("hello")
	if len(messages) < 2 {
		t.Fatalf("messages length = %d, want at least 2", len(messages))
	}
	if messages[0]["role"] != "system" || messages[0]["content"] != "   " {
		t.Fatalf("first message = %#v, want whitespace system prompt preserved", messages[0])
	}
}

func TestSimpleAgentKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}
	if agent.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", agent.MaxToolIterations)
	}
}

func TestNewSimpleAgentReturnsErrorWhenTraceLoggerInitFails(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = true
	cfg.TraceDir = filepath.Join(blocker, "traces")
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	_, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		3,
	)
	if err == nil {
		t.Fatalf("NewSimpleAgentWithOptions should return error when trace logger init fails")
	}
}

func TestSimpleAgentAddToolDefaultAutoExpandTrueLikePython(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		false,
		3,
	)
	if err != nil {
		t.Fatalf("NewSimpleAgentWithOptions error: %v", err)
	}

	agent.AddTool(newExpandableTestTool())

	if agent.ToolRegistry.GetTool("child") == nil {
		t.Fatalf("child tool should be registered when auto_expand defaults to true")
	}
	if agent.ToolRegistry.GetTool("parent") != nil {
		t.Fatalf("parent tool should not remain when auto expansion is applied")
	}
}

// ---------------------------------------------------------------------------
// SimpleAgent run tests (from agents/simple_agent_run_test.go)
// ---------------------------------------------------------------------------

func TestSimpleAgentRunWithoutTools(t *testing.T) {
	llm := testutil.NewMockLLM("你好！我是AI助手。")
	agent, err := agents.NewSimpleAgent("test-agent", llm, "你是一个AI助手", noTraceConfig(), nil)
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

	agent, err := agents.NewSimpleAgentWithOptions("test-agent", llm, "你是一个AI助手", noTraceConfig(), registry, true, 3)
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
	agent, err := agents.NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
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
	agent, err := agents.NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
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

	agent, err := agents.NewSimpleAgent("test", llm, "", noTraceConfig(), nil)
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

	full := ""
	for _, c := range chunks {
		full += c
	}
	if full != "Hello World" {
		t.Fatalf("streamed result = %q, want 'Hello World'", full)
	}
}

// ---------------------------------------------------------------------------
// ReActAgent tests (from agents/react_agent_test.go, react_agent_run_test.go)
// ---------------------------------------------------------------------------

func TestReActAgentKeepsExplicitZeroMaxSteps(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewReActAgent(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}
	if agent.MaxSteps != 0 {
		t.Fatalf("MaxSteps = %d, want 0", agent.MaxSteps)
	}
}

func TestReActAgentRunBasicFlow(t *testing.T) {
	thoughtCall := testutil.MockToolCallResponse("Thought", "call-1", map[string]any{
		"reasoning": "我需要思考这个问题",
	})
	finishCall := testutil.MockToolCallResponse("Finish", "call-2", map[string]any{
		"answer": "最终答案是42",
	})

	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{thoughtCall, finishCall},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	registry := tools.NewToolRegistry(nil)

	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, registry, 10)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	result, err := agent.Run("什么是42？", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}

func TestReActAgentRunWithMaxStepsExceeded(t *testing.T) {
	thoughtCall := testutil.MockToolCallResponse("Thought", "call-1", map[string]any{
		"reasoning": "让我继续思考",
	})

	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{
			thoughtCall, thoughtCall, thoughtCall, thoughtCall, thoughtCall,
		},
		Responses: []string{"超过最大步数的回退响应"},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	registry := tools.NewToolRegistry(nil)

	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, registry, 2)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	result, err := agent.Run("思考这个问题", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty even when max steps exceeded")
	}
}

func TestReActAgentArun(t *testing.T) {
	finishCall := testutil.MockToolCallResponse("Finish", "call-1", map[string]any{
		"answer": "直接回答",
	})
	adapter := &testutil.MockLLMAdapter{
		ToolResponses: []map[string]any{finishCall},
	}
	llm := testutil.NewMockLLMFromAdapter(adapter)

	cfg := noTraceConfig()
	agent, err := agents.NewReActAgent("react-test", llm, "", cfg, nil, 5)
	if err != nil {
		t.Fatalf("NewReActAgent error: %v", err)
	}

	var started bool
	hooks := core.Hooks{
		OnStart: func(e core.AgentEvent) error { started = true; return nil },
	}

	result, err := agent.Arun("hi", hooks, nil)
	if err != nil {
		t.Fatalf("Arun error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
	if !started {
		t.Fatal("OnStart hook was not called")
	}
}

// ---------------------------------------------------------------------------
// ReflectionAgent tests (from agents/reflection_agent_test.go, reflection_agent_run_test.go)
// ---------------------------------------------------------------------------

func TestReflectionAgentKeepsExplicitZeroIterationSettings(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewReflectionAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		0,
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewReflectionAgentWithOptions error: %v", err)
	}
	if agent.MaxIterations != 0 {
		t.Fatalf("MaxIterations = %d, want 0", agent.MaxIterations)
	}
	if agent.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", agent.MaxToolIterations)
	}
}

func TestReflectionAgentRunBasicFlow(t *testing.T) {
	llm := testutil.NewMockLLM(
		"初次执行结果",
		"反思：结果看起来不错，满意度 9/10，我对结果满意。",
		"优化后的最终结果",
	)

	cfg := noTraceConfig()
	agent, err := agents.NewReflectionAgentWithOptions("reflect-test", llm, "", cfg, 3, nil, false, 3)
	if err != nil {
		t.Fatalf("NewReflectionAgent error: %v", err)
	}

	result, err := agent.Run("写一首诗", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}

func TestReflectionAgentCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")
	cfg := noTraceConfig()

	agent, err := agents.NewReflectionAgentWithOptions("test", llm, "", cfg, 2, nil, false, 3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if agent.MaxIterations != 2 {
		t.Fatalf("MaxIterations = %d, want 2", agent.MaxIterations)
	}
}

// ---------------------------------------------------------------------------
// PlanSolveAgent tests (from agents/plan_solve_agent_test.go, plan_solve_agent_run_test.go)
// ---------------------------------------------------------------------------

func TestPlannerStepsFromArgsKeepsWhitespaceAndEmptyItems(t *testing.T) {
	args := map[string]any{
		"steps": []any{"  step one  ", "", "step three"},
	}

	got := agents.ExportPlannerStepsFromArgs(args)
	want := []string{"  step one  ", "", "step three"}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExecutorKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	executor := agents.NewExecutor(
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		nil,
		false,
		0,
	)
	if executor.MaxToolIterations != 0 {
		t.Fatalf("MaxToolIterations = %d, want 0", executor.MaxToolIterations)
	}
}

func TestPlanSolveAgentKeepsExplicitZeroMaxToolIterations(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewPlanSolveAgentWithOptions(
		"tester",
		core.NewLLMFromAdapter("test-model", "", "", 0, 0, nil),
		"",
		&cfg,
		"",
		"",
		nil,
		false,
		0,
	)
	if err != nil {
		t.Fatalf("NewPlanSolveAgentWithOptions error: %v", err)
	}
	if agent.Executor.MaxToolIterations != 0 {
		t.Fatalf("Executor.MaxToolIterations = %d, want 0", agent.Executor.MaxToolIterations)
	}
}

func TestPlanSolveAgentCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")
	cfg := noTraceConfig()

	agent, err := agents.NewPlanSolveAgentWithOptions("ps-test", llm, "", cfg, "", "", nil, false, 3)
	if err != nil {
		t.Fatalf("NewPlanSolveAgentWithOptions error: %v", err)
	}

	if agent.AgentType != "PlanSolveAgent" {
		t.Fatalf("AgentType = %q, want PlanSolveAgent", agent.AgentType)
	}
}

func TestPlannerAndExecutorCreation(t *testing.T) {
	llm := testutil.NewMockLLM("test")

	planner := agents.NewPlanner(llm, "")
	if planner == nil {
		t.Fatal("NewPlanner returned nil")
	}

	executor := agents.NewExecutor(llm, "", nil, false, 3)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}
}

func TestPlannerStepsFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected int
	}{
		{"nil kwargs", nil, 0},
		{"missing key", map[string]any{}, 0},
		{"with steps", map[string]any{"steps": []any{"step1", "step2"}}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agents.ExportPlannerStepsFromArgs(tt.input)
			if len(got) != tt.expected {
				t.Fatalf("len(plannerStepsFromArgs()) = %d, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestPlanSolveAgentRunBasicFlow(t *testing.T) {
	llm := testutil.NewMockLLM(
		"步骤1: 分析问题\n步骤2: 收集信息\n步骤3: 生成答案",
		"执行步骤1的结果",
		"执行步骤2的结果",
		"执行步骤3的结果：最终答案是42",
	)

	cfg := noTraceConfig()
	agent, err := agents.NewPlanSolveAgentWithOptions("ps-test", llm, "", cfg, "", "", nil, false, 3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	result, err := agent.Run("解决这个复杂问题", nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if result == "" {
		t.Fatal("result should not be empty")
	}
}
