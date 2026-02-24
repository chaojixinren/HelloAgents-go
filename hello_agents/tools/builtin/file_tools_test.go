package builtin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"helloagents-go/hello_agents/tools"
)

func TestWriteEditMultiEditResolvePathNormalizesBackslashes(t *testing.T) {
	root := t.TempDir()

	writeTool := NewWriteToolWithOptions(root, root, nil)
	editTool := NewEditToolWithOptions(root, root, nil)
	multiEditTool := NewMultiEditToolWithOptions(root, root, nil)

	want := filepath.Clean(filepath.Join(root, "nested", "file.txt"))

	if got := filepath.Clean(writeTool.resolvePath(`nested\file.txt`)); got != want {
		t.Fatalf("WriteTool.resolvePath() = %q, want %q", got, want)
	}
	if got := filepath.Clean(editTool.resolvePath(`nested\file.txt`)); got != want {
		t.Fatalf("EditTool.resolvePath() = %q, want %q", got, want)
	}
	if got := filepath.Clean(multiEditTool.resolvePath(`nested\file.txt`)); got != want {
		t.Fatalf("MultiEditTool.resolvePath() = %q, want %q", got, want)
	}
}

func TestWriteToolTreatsProvidedZeroMtimeAsConflictCheckInput(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "a.txt")
	if err := os.WriteFile(filePath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write seed file error: %v", err)
	}

	tool := NewWriteToolWithOptions(root, root, nil)
	resp := tool.Run(map[string]any{
		"path":          "a.txt",
		"content":       "new",
		"file_mtime_ms": 0,
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeConflict {
		t.Fatalf("error code = %v, want %q", resp.ErrorInfo, tools.ToolErrorCodeConflict)
	}
}

func TestReadToolLimitZeroMeansNoLimit(t *testing.T) {
	root := t.TempDir()
	lines := make([]string, 0, 2501)
	for i := 0; i < 2501; i++ {
		lines = append(lines, "line")
	}

	filePath := filepath.Join(root, "big.txt")
	if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write seed file error: %v", err)
	}

	tool := NewReadToolWithOptions(root, root, nil)
	resp := tool.Run(map[string]any{
		"path":  "big.txt",
		"limit": 0,
	})

	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusSuccess)
	}
	gotLines, _ := resp.Data["lines"].(int)
	if gotLines != 2501 {
		t.Fatalf("lines = %d, want 2501 when limit=0", gotLines)
	}
}

func TestReadToolHandlesTrailingNewlineLikePythonReadlines(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "t.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("write seed file error: %v", err)
	}

	tool := NewReadToolWithOptions(root, root, nil)
	resp := tool.Run(map[string]any{"path": "t.txt"})
	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusSuccess)
	}

	totalLines, _ := resp.Data["total_lines"].(int)
	if totalLines != 2 {
		t.Fatalf("total_lines = %d, want 2", totalLines)
	}

	content, _ := resp.Data["content"].(string)
	if content != "a\nb\n" {
		t.Fatalf("content = %q, want %q", content, "a\nb\n")
	}
}

func TestWriteToolReturnsPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod permission semantics differ on windows")
	}

	root := t.TempDir()
	lockedDir := filepath.Join(root, "locked")
	if err := os.MkdirAll(lockedDir, 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.Chmod(lockedDir, 0o500); err != nil {
		t.Fatalf("chmod error: %v", err)
	}
	defer os.Chmod(lockedDir, 0o755)

	tool := NewWriteToolWithOptions(root, root, nil)
	resp := tool.Run(map[string]any{
		"path":    "locked/a.txt",
		"content": "x",
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodePermissionDenied {
		t.Fatalf("error = %#v, want PERMISSION_DENIED", resp.ErrorInfo)
	}
}

func TestEditAndMultiEditReturnPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod permission semantics differ on windows")
	}

	root := t.TempDir()
	filePath := filepath.Join(root, "locked.txt")
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("seed file write error: %v", err)
	}
	if err := os.Chmod(filePath, 0o400); err != nil {
		t.Fatalf("chmod file error: %v", err)
	}
	defer os.Chmod(filePath, 0o644)

	editTool := NewEditToolWithOptions(root, root, nil)
	editResp := editTool.Run(map[string]any{
		"path":       "locked.txt",
		"old_string": "a",
		"new_string": "z",
	})
	if editResp.Status != tools.ToolStatusError {
		t.Fatalf("edit status = %q, want %q", editResp.Status, tools.ToolStatusError)
	}
	if editResp.ErrorInfo == nil || editResp.ErrorInfo["code"] != tools.ToolErrorCodePermissionDenied {
		t.Fatalf("edit error = %#v, want PERMISSION_DENIED", editResp.ErrorInfo)
	}

	multiEditTool := NewMultiEditToolWithOptions(root, root, nil)
	multiResp := multiEditTool.Run(map[string]any{
		"path": "locked.txt",
		"edits": []any{
			map[string]any{"old_string": "a", "new_string": "z"},
		},
	})
	if multiResp.Status != tools.ToolStatusError {
		t.Fatalf("multiedit status = %q, want %q", multiResp.Status, tools.ToolStatusError)
	}
	if multiResp.ErrorInfo == nil || multiResp.ErrorInfo["code"] != tools.ToolErrorCodePermissionDenied {
		t.Fatalf("multiedit error = %#v, want PERMISSION_DENIED", multiResp.ErrorInfo)
	}
}
