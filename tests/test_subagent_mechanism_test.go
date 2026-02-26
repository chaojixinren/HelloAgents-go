package tests_test

import (
	"fmt"
	"testing"

	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type noopTool struct {
	tools.BaseTool
}

func newNoopTool(name string) *noopTool {
	base := tools.NewBaseTool(name, "noop", false)
	return &noopTool{BaseTool: base}
}

func (t *noopTool) Run(parameters map[string]any) tools.ToolResponse {
	return tools.Success("ok", nil, nil, nil)
}

type fakeSubagentRunner struct {
	seenMaxSteps *int
}

func (f *fakeSubagentRunner) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	f.seenMaxSteps = maxStepsOverride
	return map[string]any{
		"success": true,
		"summary": "ok",
		"metadata": map[string]any{
			"steps": 1,
		},
	}
}

type panicSubagentRunner struct{}

func (p *panicSubagentRunner) RunAsSubagent(task string, toolFilter tools.ToolFilter, returnSummary bool, maxStepsOverride *int) map[string]any {
	panic("boom")
}

// ---------------------------------------------------------------------------
// TaskTool tests (from tools/builtin/task_tool_test.go)
// ---------------------------------------------------------------------------

func TestTaskToolPassesZeroMaxStepsOverride(t *testing.T) {
	runner := &fakeSubagentRunner{}
	tool := builtin.NewTaskTool(func(agentType string) (any, error) {
		return runner, nil
	}, nil)

	response := tool.Run(map[string]any{
		"task":      "do something",
		"max_steps": 0,
	})
	if response.Status != tools.ToolStatusSuccess {
		t.Fatalf("TaskTool.Run() status = %v, want %v", response.Status, tools.ToolStatusSuccess)
	}
	if runner.seenMaxSteps == nil {
		t.Fatalf("TaskTool.Run() did not pass max_steps override")
	}
	if *runner.seenMaxSteps != 0 {
		t.Fatalf("TaskTool.Run() max_steps = %d, want 0", *runner.seenMaxSteps)
	}

	response = tool.Run(map[string]any{
		"task":      "do something",
		"max_steps": true,
	})
	if response.Status != tools.ToolStatusSuccess {
		t.Fatalf("TaskTool.Run() status = %v, want %v", response.Status, tools.ToolStatusSuccess)
	}
	if runner.seenMaxSteps == nil || *runner.seenMaxSteps != 1 {
		t.Fatalf("TaskTool.Run() bool max_steps should map to 1, got %#v", runner.seenMaxSteps)
	}
}

func TestTaskToolReturnsExecutionErrorOnPanic(t *testing.T) {
	tool := builtin.NewTaskTool(func(agentType string) (any, error) {
		return &panicSubagentRunner{}, nil
	}, nil)

	response := tool.Run(map[string]any{
		"task": "panic case",
	})
	if response.Status != tools.ToolStatusError {
		t.Fatalf("TaskTool.Run() status = %v, want %v", response.Status, tools.ToolStatusError)
	}
	if response.ErrorInfo == nil || response.ErrorInfo["code"] != tools.ToolErrorCodeExecutionError {
		t.Fatalf("TaskTool.Run() error code = %#v, want %s", response.ErrorInfo, tools.ToolErrorCodeExecutionError)
	}
}

func TestTaskToolDoesNotTrimWhitespaceTask(t *testing.T) {
	runner := &fakeSubagentRunner{}
	tool := builtin.NewTaskTool(func(agentType string) (any, error) {
		return runner, nil
	}, nil)

	response := tool.Run(map[string]any{
		"task": "   ",
	})
	if response.Status != tools.ToolStatusSuccess {
		t.Fatalf("TaskTool.Run() status = %v, want %v", response.Status, tools.ToolStatusSuccess)
	}
}

func TestTaskToolDefaultsOnlyWhenAgentTypeIsMissing(t *testing.T) {
	runner := &fakeSubagentRunner{}
	tool := builtin.NewTaskTool(func(agentType string) (any, error) {
		if agentType != "react" {
			return nil, fmt.Errorf("unsupported %s", agentType)
		}
		return runner, nil
	}, nil)

	missing := tool.Run(map[string]any{
		"task": "x",
	})
	if missing.Status != tools.ToolStatusSuccess {
		t.Fatalf("missing agent_type should use default react, got status %v", missing.Status)
	}

	explicitEmpty := tool.Run(map[string]any{
		"task":       "x",
		"agent_type": "",
	})
	if explicitEmpty.Status != tools.ToolStatusError {
		t.Fatalf("explicit empty agent_type should not fallback to default, got %v", explicitEmpty.Status)
	}
	if explicitEmpty.ErrorInfo == nil || explicitEmpty.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("error = %#v, want INVALID_PARAM", explicitEmpty.ErrorInfo)
	}
}

// ---------------------------------------------------------------------------
// BaseAgent subagent tests (from core/agent_test.go)
// ---------------------------------------------------------------------------

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
