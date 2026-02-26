package context

import "testing"

func TestTruncatorDoesNotByteTrimPreviewAfterLineTruncation(t *testing.T) {
	truncator := NewObservationTruncator(10, 5, "head", t.TempDir())
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
	truncator := NewObservationTruncator(2, 1024, "head", t.TempDir())
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
	truncator := NewObservationTruncator(0, 0, "head", t.TempDir())
	if truncator.MaxLines != 0 {
		t.Fatalf("MaxLines = %d, want 0", truncator.MaxLines)
	}
	if truncator.MaxBytes != 0 {
		t.Fatalf("MaxBytes = %d, want 0", truncator.MaxBytes)
	}
}

func TestTruncatorEmptyOutputDirDoesNotPanic(t *testing.T) {
	tr := NewObservationTruncator(10, 10, "head", "")
	if tr == nil {
		t.Fatalf("NewObservationTruncator returned nil for empty output_dir")
	}
}
