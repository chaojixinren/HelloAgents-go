package observability

import (
	"os"
	"testing"
)

func TestComputeStatsTracksStepTokenAndToolCallMetrics(t *testing.T) {
	logger, err := NewTraceLogger(t.TempDir(), false, false)
	if err != nil {
		t.Fatalf("NewTraceLogger() error = %v", err)
	}

	step1 := 1
	step2 := 2
	step3 := 3

	logger.LogEvent("session_start", map[string]any{"agent_name": "tester"}, nil)
	logger.LogEvent("tool_call", map[string]any{"tool_name": "Read"}, &step1)
	logger.LogEvent("tool_call", map[string]any{"tool_name": "   "}, &step1)
	logger.LogEvent("tool_call", map[string]any{}, &step2)
	logger.LogEvent("model_output", map[string]any{
		"usage": map[string]any{
			"total_tokens": 12,
			"cost":         0.5,
		},
	}, &step3)
	logger.LogEvent("error", map[string]any{"error_type": "E", "message": "boom"}, &step2)
	logger.LogEvent("session_end", map[string]any{}, nil)

	stats := logger.computeStats()

	if got := stats["total_steps"]; got != 3 {
		t.Fatalf("total_steps = %v, want 3", got)
	}
	if got := stats["total_tokens"]; got != 12 {
		t.Fatalf("total_tokens = %v, want 12", got)
	}
	if got := stats["model_calls"]; got != 1 {
		t.Fatalf("model_calls = %v, want 1", got)
	}

	toolCalls, ok := stats["tool_calls"].(map[string]int)
	if !ok {
		t.Fatalf("tool_calls type = %T, want map[string]int", stats["tool_calls"])
	}
	if got := toolCalls["Read"]; got != 1 {
		t.Fatalf("tool_calls[Read] = %d, want 1", got)
	}
	if got := toolCalls["   "]; got != 1 {
		t.Fatalf("tool_calls[\"   \"] = %d, want 1", got)
	}
	if got := toolCalls["unknown"]; got != 1 {
		t.Fatalf("tool_calls[unknown] = %d, want 1", got)
	}

	errors, ok := stats["errors"].([]map[string]any)
	if !ok {
		t.Fatalf("errors type = %T, want []map[string]any", stats["errors"])
	}
	if len(errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(errors))
	}

	if err := logger.Finalize(); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
}

func TestNewTraceLoggerEmptyOutputDirUsesCurrentDirectoryLikePathlib(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
	})

	logger, err := NewTraceLogger("", false, false)
	if err != nil {
		t.Fatalf("NewTraceLogger() error = %v", err)
	}
	if logger.OutputDir != "." {
		t.Fatalf("OutputDir = %q, want %q", logger.OutputDir, ".")
	}
	_ = logger.Finalize()
}

func TestParseTraceTimestampSupportsPythonISO(t *testing.T) {
	if _, err := parseTraceTimestamp("2026-02-24T12:34:56.123456"); err != nil {
		t.Fatalf("parseTraceTimestamp() error = %v", err)
	}
}
