package tests_test

import (
	"testing"

	"helloagents-go/hello_agents/context"
)

// ---------------------------------------------------------------------------
// HistoryManager tests (from context/history_test.go)
// ---------------------------------------------------------------------------

func TestHistoryManagerAppendAndGet(t *testing.T) {
	hm := context.NewHistoryManager[string](
		2,
		0.8,
		func(summary string) string { return "[summary] " + summary },
		func(msg string) string { return msg },
	)

	hm.Append("msg1")
	hm.Append("msg2")
	hm.Append("msg3")

	history := hm.GetHistory()
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}
}

func TestHistoryManagerClear(t *testing.T) {
	hm := context.NewHistoryManager[string](
		2, 0.8,
		func(summary string) string { return summary },
		func(msg string) string { return msg },
	)

	hm.Append("msg1")
	hm.Clear()

	if len(hm.GetHistory()) != 0 {
		t.Fatal("history should be empty after Clear()")
	}
}

func TestHistoryManagerEstimateRounds(t *testing.T) {
	hm := context.NewHistoryManager[string](
		2, 0.8,
		func(summary string) string { return summary },
		func(msg string) string {
			if msg == "user-msg" {
				return "user"
			}
			return "assistant"
		},
	)

	hm.Append("user-msg")
	hm.Append("assistant-msg")
	hm.Append("user-msg")
	hm.Append("assistant-msg")

	rounds := hm.EstimateRounds()
	if rounds != 2 {
		t.Fatalf("EstimateRounds() = %d, want 2", rounds)
	}
}

func TestHistoryManagerCompress(t *testing.T) {
	hm := context.NewHistoryManager[string](
		1, 0.8,
		func(summary string) string { return "[compressed] " + summary },
		func(msg string) string {
			if msg == "user" {
				return "user"
			}
			return "assistant"
		},
	)

	hm.Append("user")
	hm.Append("assistant")
	hm.Append("user")
	hm.Append("assistant")

	hm.Compress("summary of old messages")

	history := hm.GetHistory()
	if len(history) == 0 {
		t.Fatal("history should not be empty after compress")
	}
}

// ---------------------------------------------------------------------------
// TokenCounter tests (from context/history_test.go)
// ---------------------------------------------------------------------------

func TestTokenCounterCountMessage(t *testing.T) {
	tc := context.NewTokenCounter[string](
		"gpt-3.5-turbo",
		func(msg string) string { return msg },
		func(msg string) string { return msg },
	)

	count := tc.CountMessage("hello world")
	if count <= 0 {
		t.Fatalf("CountMessage returned %d, want > 0", count)
	}
}

func TestTokenCounterCountMessages(t *testing.T) {
	tc := context.NewTokenCounter[string](
		"gpt-3.5-turbo",
		func(msg string) string { return msg },
		func(msg string) string { return msg },
	)

	messages := []string{"hello", "world", "test"}
	count := tc.CountMessages(messages)
	if count <= 0 {
		t.Fatalf("CountMessages returned %d, want > 0", count)
	}
}

// ---------------------------------------------------------------------------
// Parity tests (from context/parity_test.go)
// ---------------------------------------------------------------------------

func TestNewHistoryManagerKeepsExplicitValues(t *testing.T) {
	h := context.NewHistoryManager[int](0, 0, nil, nil)
	if h.MinRetainRounds != 0 {
		t.Fatalf("MinRetainRounds = %d, want 0", h.MinRetainRounds)
	}
	if h.CompressionThreshold != 0 {
		t.Fatalf("CompressionThreshold = %v, want 0", h.CompressionThreshold)
	}
}

func TestNewTokenCounterKeepsEmptyModel(t *testing.T) {
	c := context.NewTokenCounter[string]("", func(s string) string { return s }, func(s string) string { return s })
	if c.Model != "" {
		t.Fatalf("Model = %q, want empty string", c.Model)
	}
}

func TestNewContextBuilderDoesNotNormalizeProvidedConfig(t *testing.T) {
	cfg := context.ContextConfig{
		MaxTokens:         0,
		ReserveRatio:      2,
		MinRelevance:      -1,
		MMRLambda:         2,
		EnableCompression: true,
	}
	builder := context.NewContextBuilder(cfg)
	if builder.Config.MaxTokens != 0 {
		t.Fatalf("MaxTokens = %d, want 0", builder.Config.MaxTokens)
	}
	if builder.Config.ReserveRatio != 2 {
		t.Fatalf("ReserveRatio = %v, want 2", builder.Config.ReserveRatio)
	}
	if builder.Config.MinRelevance != -1 {
		t.Fatalf("MinRelevance = %v, want -1", builder.Config.MinRelevance)
	}
	if builder.Config.MMRLambda != 2 {
		t.Fatalf("MMRLambda = %v, want 2", builder.Config.MMRLambda)
	}
}

// ---------------------------------------------------------------------------
// ObservationTruncator tests (from context/truncator_test.go)
// ---------------------------------------------------------------------------

func TestTruncatorDoesNotByteTrimPreviewAfterLineTruncation(t *testing.T) {
	truncator := context.NewObservationTruncator(10, 5, "head", t.TempDir())
	content := "abcdefghijklmnopqrstuvwxyz"

	preview, result := truncator.TruncateWithMetadata(content, "ToolX", nil)
	truncated, _ := result["truncated"].(bool)
	if !truncated {
		t.Fatalf("truncated = false, want true when bytes exceed limit")
	}
	if preview != content {
		t.Fatalf("preview = %q, want full line preview %q (python behavior)", preview, content)
	}
}

func TestTruncatorSplitLinesMatchesPythonSplitlines(t *testing.T) {
	truncator := context.NewObservationTruncator(2, 1024, "head", t.TempDir())
	content := "a\nb\n"

	preview, result := truncator.TruncateWithMetadata(content, "ToolX", nil)
	truncated, _ := result["truncated"].(bool)
	if truncated {
		t.Fatalf("truncated = true, want false for two logical lines with trailing newline")
	}
	if preview != content {
		t.Fatalf("preview = %q, want original content %q when not truncated", preview, content)
	}
}

func TestTruncatorKeepsExplicitZeroLimits(t *testing.T) {
	truncator := context.NewObservationTruncator(0, 0, "head", t.TempDir())
	if truncator.MaxLines != 0 {
		t.Fatalf("MaxLines = %d, want 0", truncator.MaxLines)
	}
	if truncator.MaxBytes != 0 {
		t.Fatalf("MaxBytes = %d, want 0", truncator.MaxBytes)
	}
}

func TestTruncatorEmptyOutputDirDoesNotPanic(t *testing.T) {
	tr := context.NewObservationTruncator(10, 10, "head", "")
	if tr == nil {
		t.Fatalf("NewObservationTruncator returned nil for empty output_dir")
	}
}
