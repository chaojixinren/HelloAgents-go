package builtin_test

import (
	"fmt"
	"testing"

	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

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
