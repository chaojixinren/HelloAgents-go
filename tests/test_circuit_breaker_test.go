package tests_test

import (
	"testing"
	"time"

	"helloagents-go/hello_agents/tools"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type dummyCircuitTool struct {
	tools.BaseTool
	shouldFail bool
	callCount  int
}

func newDummyCircuitTool(name string, shouldFail bool) *dummyCircuitTool {
	base := tools.NewBaseTool(name, "dummy circuit tool", false)
	return &dummyCircuitTool{BaseTool: base, shouldFail: shouldFail}
}

func (t *dummyCircuitTool) Run(parameters map[string]any) tools.ToolResponse {
	t.callCount++
	if t.shouldFail {
		return tools.Error("Dummy tool failed", tools.ToolErrorCodeExecutionError, nil)
	}
	return tools.Success("Success", nil, nil, nil)
}

func (t *dummyCircuitTool) RunWithTiming(parameters map[string]any) tools.ToolResponse {
	return t.Run(parameters)
}

// ---------------------------------------------------------------------------
// CircuitBreaker tests (from tools/circuit_breaker_test.go)
// ---------------------------------------------------------------------------

func TestCircuitBreakerFailureThreshold(t *testing.T) {
	cb := tools.NewCircuitBreaker(3, 300, true)
	for i := 0; i < 3; i++ {
		cb.RecordResult("test_tool", tools.Error("boom", tools.ToolErrorCodeExecutionError, nil))
	}
	if !cb.IsOpen("test_tool") {
		t.Fatalf("test_tool should be open after reaching failure threshold")
	}
	status := cb.GetStatus("test_tool")
	if status["state"] != "open" {
		t.Fatalf("state = %v, want open", status["state"])
	}
	if status["failure_count"] != 3 {
		t.Fatalf("failure_count = %v, want 3", status["failure_count"])
	}
}

func TestCircuitBreakerKeepsExplicitZeroConfigValues(t *testing.T) {
	cb := tools.NewCircuitBreaker(0, 0, true)
	if cb.FailureThreshold != 0 {
		t.Fatalf("FailureThreshold = %d, want 0", cb.FailureThreshold)
	}
	if cb.RecoveryTimeout != 0 {
		t.Fatalf("RecoveryTimeout = %d, want 0", cb.RecoveryTimeout)
	}
}

func TestCircuitBreakerSuccessResetsCounter(t *testing.T) {
	cb := tools.NewCircuitBreaker(3, 300, true)
	cb.RecordResult("test_tool", tools.Error("boom", tools.ToolErrorCodeExecutionError, nil))
	cb.RecordResult("test_tool", tools.Error("boom", tools.ToolErrorCodeExecutionError, nil))
	cb.RecordResult("test_tool", tools.Success("ok", nil, nil, nil))

	status := cb.GetStatus("test_tool")
	if status["failure_count"] != 0 {
		t.Fatalf("failure_count = %v, want 0 after success", status["failure_count"])
	}
	if status["state"] != "closed" {
		t.Fatalf("state = %v, want closed", status["state"])
	}
}

func TestCircuitBreakerAutoRecovery(t *testing.T) {
	cb := tools.NewCircuitBreaker(2, 1, true)
	cb.RecordResult("test_tool", tools.Error("boom", tools.ToolErrorCodeExecutionError, nil))
	cb.RecordResult("test_tool", tools.Error("boom", tools.ToolErrorCodeExecutionError, nil))
	if !cb.IsOpen("test_tool") {
		t.Fatalf("test_tool should be open before recovery timeout")
	}

	time.Sleep(1100 * time.Millisecond)
	if cb.IsOpen("test_tool") {
		t.Fatalf("test_tool should auto-recover after timeout")
	}
}

func TestToolRegistryCircuitBreakerBlocksAndRecovers(t *testing.T) {
	cb := tools.NewCircuitBreaker(2, 1, true)
	registry := tools.NewToolRegistry(cb)
	tool := newDummyCircuitTool("dummy_tool", true)
	registry.RegisterTool(tool, false)

	for i := 0; i < 2; i++ {
		resp := registry.ExecuteTool("dummy_tool", "x")
		if resp.Status != tools.ToolStatusError {
			t.Fatalf("status = %q, want error", resp.Status)
		}
	}

	blocked := registry.ExecuteTool("dummy_tool", "x")
	if blocked.ErrorInfo == nil || blocked.ErrorInfo["code"] != tools.ToolErrorCodeCircuitOpen {
		t.Fatalf("blocked error code = %v, want %q", blocked.ErrorInfo, tools.ToolErrorCodeCircuitOpen)
	}
	if tool.callCount != 2 {
		t.Fatalf("tool callCount = %d, want 2 when circuit is open", tool.callCount)
	}

	time.Sleep(1100 * time.Millisecond)
	retried := registry.ExecuteTool("dummy_tool", "x")
	if retried.ErrorInfo == nil || retried.ErrorInfo["code"] == tools.ToolErrorCodeCircuitOpen {
		t.Fatalf("after recovery, should execute tool instead of returning CIRCUIT_OPEN")
	}
	if tool.callCount != 3 {
		t.Fatalf("tool callCount = %d, want 3 after recovery retry", tool.callCount)
	}
}
