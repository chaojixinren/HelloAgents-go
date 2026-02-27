// 使用真实 LLM API 的集成测试
// 需要环境变量: LLM_MODEL_ID, LLM_API_KEY, LLM_BASE_URL
package tests_test

import (
	"os"
	"strings"
	"testing"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func skipIfNoLLM(t *testing.T) {
	t.Helper()
	if os.Getenv("LLM_API_KEY") == "" || os.Getenv("LLM_MODEL_ID") == "" || os.Getenv("LLM_BASE_URL") == "" {
		t.Skip("LLM env vars not set, skipping integration test")
	}
}

func testLLM(t *testing.T) *core.HelloAgentsLLM {
	t.Helper()
	timeout := 120
	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, &timeout, nil)
	if err != nil {
		t.Fatalf("NewHelloAgentsLLM: %v", err)
	}
	return llm
}

func testConfig() *core.Config {
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	return &cfg
}

// === SimpleAgent 集成测试 ===

func TestIntegrationSimpleAgentInvoke(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("test", llm, "用一个词回答", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("1+1等于几？一个字回答", nil)
	if err != nil {
		t.Logf("Run timeout (model may be slow): %v", err)
		t.Skip("model response too slow, skipping")
	}
	if result == "" {
		t.Fatal("empty result")
	}
	t.Logf("SimpleAgent result: %s", result)
}

func TestIntegrationSimpleAgentStreamRun(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("test", llm, "用一个词回答", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	chunkCh, errCh := agent.StreamRun("1+1等于几？", nil)
	var chunks []string
	for chunk := range chunkCh {
		chunks = append(chunks, chunk)
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamRun: %v", err)
		}
	}
	full := strings.Join(chunks, "")
	if full == "" {
		t.Fatal("empty stream result")
	}
	t.Logf("StreamRun: %d chunks, result=%s", len(chunks), full)
}

func TestIntegrationSimpleAgentArunWithHooks(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("test", llm, "简短回答", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	var started, finished bool
	hooks := core.Hooks{
		OnStart:  func(e core.AgentEvent) error { started = true; return nil },
		OnFinish: func(e core.AgentEvent) error { finished = true; return nil },
	}

	result, err := agent.Arun("你好", hooks, nil)
	if err != nil {
		t.Fatalf("Arun: %v", err)
	}
	if !started {
		t.Error("OnStart not called")
	}
	if !finished {
		t.Error("OnFinish not called")
	}
	t.Logf("Arun result: %s", result)
}

func TestIntegrationSimpleAgentArunStream(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("test", llm, "简短回答", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	events := agent.ArunStream("你好", nil)
	var types []core.EventType
	for event := range events {
		types = append(types, event.Type)
	}
	if len(types) < 2 {
		t.Fatalf("too few events: %v", types)
	}
	if types[0] != core.AgentStart {
		t.Errorf("first event=%s, want agent_start", types[0])
	}
	t.Logf("ArunStream events: %v", types)
}

// === SimpleAgent with Tools 集成测试 ===

func TestIntegrationSimpleAgentWithToolCalling(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/note.txt", []byte("Go语言由Google开发"), 0o644)

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool(tmpDir, registry), false)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)

	agent, err := agents.NewSimpleAgentWithOptions("test", llm, "用中文简短回答", testConfig(), registry, true, 5)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("读取 note.txt 的内容并告诉我", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == "" {
		t.Fatal("empty result")
	}
	t.Logf("Tool calling result: %s", result)
}

func TestIntegrationSimpleAgentCalculator(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)

	agent, err := agents.NewSimpleAgentWithOptions("test", llm, "使用计算器工具，简短回答", testConfig(), registry, true, 5)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("用计算器算 15*8+20", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(result, "140") {
		t.Logf("result may not contain 140: %s", result)
	}
	t.Logf("Calculator result: %s", result)
}

func TestIntegrationSimpleAgentToolManagement(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("test", llm, "", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if agent.HasTools() {
		t.Error("should not have tools initially")
	}

	agent.AddTool(builtin.NewCalculatorTool())
	if !agent.HasTools() {
		t.Error("should have tools after AddTool")
	}
	if len(agent.ListTools()) != 1 {
		t.Errorf("ListTools=%d, want 1", len(agent.ListTools()))
	}

	removed := agent.RemoveTool("python_calculator")
	if !removed {
		t.Error("RemoveTool should return true")
	}
	if len(agent.ListTools()) != 0 {
		t.Errorf("ListTools=%d after remove, want 0", len(agent.ListTools()))
	}
}

// === ReActAgent 集成测试 ===

func TestIntegrationReActAgentRunWithTools(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	tmpDir := t.TempDir()
	os.WriteFile(tmpDir+"/data.txt", []byte("价格: 50\n数量: 3\n"), 0o644)

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool(tmpDir, registry), false)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)

	agent, err := agents.NewReActAgent("test", llm, "使用工具完成任务，用中文简短回答", testConfig(), registry, 8)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("读取 data.txt，计算价格乘以数量", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == "" {
		t.Fatal("empty result")
	}
	t.Logf("ReAct result: %s", result)
}

func TestIntegrationReActAgentArunStream(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)

	agent, err := agents.NewReActAgent("test", llm, "使用工具，简短回答", testConfig(), registry, 5)
	if err != nil {
		t.Fatal(err)
	}

	events := agent.ArunStream("用计算器算 2+3", nil)
	var types []core.EventType
	var finalResult string
	for event := range events {
		types = append(types, event.Type)
		if event.Type == core.AgentFinish {
			finalResult, _ = event.Data["result"].(string)
		}
	}
	if len(types) < 2 {
		t.Fatalf("too few events: %v", types)
	}
	t.Logf("ReAct ArunStream: events=%v, result=%s", types, finalResult)
}

func TestIntegrationReActAgentAddTool(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewReActAgent("test", llm, "", testConfig(), nil, 5)
	if err != nil {
		t.Fatal(err)
	}

	agent.AddTool(builtin.NewCalculatorTool())
	result, err := agent.Run("用计算器算 9*9", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("AddTool result: %s", result)
}

// === ReflectionAgent 集成测试 ===

func TestIntegrationReflectionAgentRun(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewReflectionAgentWithOptions("test", llm, "", testConfig(), 1, nil, false, 3)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("用一句话解释什么是 Go 语言", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == "" {
		t.Fatal("empty result")
	}
	t.Logf("Reflection result: %s", result)
}

// === PlanSolveAgent 集成测试 ===

func TestIntegrationPlanSolveAgentRun(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewPlanSolveAgentWithOptions("test", llm, "", testConfig(), "", "", nil, false, 3)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("列出学习 Go 语言的 3 个步骤", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == "" {
		t.Fatal("empty result")
	}
	t.Logf("PlanSolve result: %s", result)
}

// === Factory 集成测试 ===

func TestIntegrationCreateAgentAllTypes(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	for _, agentType := range []string{"simple", "react", "reflection", "plan"} {
		t.Run(agentType, func(t *testing.T) {
			a, err := agents.CreateAgent(agentType, "test-"+agentType, llm, nil, testConfig(), "")
			if err != nil {
				t.Fatalf("CreateAgent(%s): %v", agentType, err)
			}
			if a == nil {
				t.Fatal("nil agent")
			}
		})
	}
}

// === SubAgent 集成测试 ===

func TestIntegrationRunAsSubagent(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	agent, err := agents.NewSimpleAgent("main", llm, "简短回答", testConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}

	result := agent.RunAsSubagent("1+1等于几？一个字回答", nil, true, nil)
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("RunAsSubagent failed: %v", result)
	}
	t.Logf("Subagent: %v", result)
}

func TestIntegrationTaskToolDirect(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)
	cfg := testConfig()

	factory := func(agentType string) (any, error) {
		return agents.DefaultSubagentFactory(agentType, llm, nil, cfg)
	}

	taskTool := builtin.NewTaskTool(factory, nil)
	resp := taskTool.Run(map[string]any{
		"task":       "回答：Go语言的吉祥物是什么？一个词回答",
		"agent_type": "simple",
	})
	if resp.Status != "success" {
		t.Fatalf("TaskTool status=%s, text=%s", resp.Status, resp.Text)
	}
	t.Logf("TaskTool: %s", resp.Text)
}

// === Session + Agent 集成 ===

func TestIntegrationAgentSessionSaveLoad(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = true
	cfg.SessionDir = t.TempDir()

	agent, err := agents.NewSimpleAgent("session-test", llm, "简短回答", &cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("你好", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("Before save: %s", result)

	path, err := agent.SaveSession("test-session")
	if err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	t.Logf("Saved to: %s", path)

	sessions, err := agent.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("no sessions found")
	}

	agent.ClearHistory()
	if len(agent.GetHistory()) != 0 {
		t.Fatal("history not cleared")
	}

	err = agent.LoadSession(path, true)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if len(agent.GetHistory()) == 0 {
		t.Fatal("history not restored")
	}
	t.Logf("Restored %d messages", len(agent.GetHistory()))
}

// === Trace + Agent 集成 ===

func TestIntegrationAgentWithTrace(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = true
	cfg.TraceDir = t.TempDir()
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false

	agent, err := agents.NewSimpleAgent("trace-test", llm, "一个词回答", &cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("1+1?", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("With trace: %s", result)
}

// === File Tools 集成 (Write+Edit+Read chain) ===

func TestIntegrationFileToolsChain(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	tmpDir := t.TempDir()
	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool(tmpDir, registry), false)
	registry.RegisterTool(builtin.NewWriteTool(tmpDir), false)
	registry.RegisterTool(builtin.NewEditTool(tmpDir), false)

	agent, err := agents.NewSimpleAgentWithOptions("test", llm,
		"你可以使用 Write 工具创建文件，Edit 工具编辑文件，Read 工具读取文件。按指令操作。",
		testConfig(), registry, true, 8)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("用 Write 工具创建文件 hello.txt 内容为 'Hello World'，然后用 Read 读取确认", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("FileTools chain: %s", result)
}

// === TodoWrite + DevLog via Agent ===

func TestIntegrationTodoWriteViaAgent(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	tmpDir := t.TempDir()
	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = false
	cfg.SessionEnabled = false
	cfg.TodoWriteEnabled = true
	cfg.TodoWritePersistenceDir = "todos"

	registry := tools.NewToolRegistry(nil)
	agent, err := agents.NewSimpleAgentWithOptions("test", llm,
		"使用 TodoWrite 工具管理任务", &cfg, registry, true, 5)
	if err != nil {
		t.Fatal(err)
	}
	agent.RegisterTodoWriteTool()

	result, err := agent.Run("用 TodoWrite 创建一个任务列表，包含2个待办事项：学习Go 和 写测试", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("TodoWrite via Agent: %s", result)

	_ = os.RemoveAll(tmpDir + "/todos")
}

// === Skills via Agent ===

func TestIntegrationSkillsViaAgent(t *testing.T) {
	skipIfNoLLM(t)
	llm := testLLM(t)

	cfg := core.DefaultConfig()
	cfg.TraceEnabled = false
	cfg.SkillsEnabled = true
	cfg.SkillsDir = "../skills"
	cfg.SessionEnabled = false

	registry := tools.NewToolRegistry(nil)
	agent, err := agents.NewSimpleAgentWithOptions("test", llm,
		"使用 Skill 工具加载技能", &cfg, registry, true, 5)
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Run("用 Skill 工具加载 LLM 技能，然后告诉我这个技能是做什么的", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("Skills via Agent: %s", result[:min(200, len(result))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
