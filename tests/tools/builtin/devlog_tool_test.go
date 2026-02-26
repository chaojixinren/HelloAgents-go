package builtin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func TestDevLogAppendAllowsWhitespaceContentLikePython(t *testing.T) {
	tool := builtin.NewDevLogTool("session-1", "Agent", t.TempDir(), "memory/devlogs")
	resp := tool.Run(map[string]any{
		"action":   "append",
		"category": "decision",
		"content":  "   ",
	})

	if resp.Status != tools.ToolStatusSuccess {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusSuccess)
	}
}

func TestDevLogSummaryCategoryOrderFollowsFirstAppearance(t *testing.T) {
	store := builtin.NewDevLogStore("session-1", "Agent")
	store.Append(builtin.NewDevLogEntry("issue", "first", nil))
	store.Append(builtin.NewDevLogEntry("decision", "second", nil))

	summary := store.GenerateSummary(10)
	if !strings.Contains(summary, "分类: issue(1), decision(1)") {
		t.Fatalf("summary = %q, want insertion-order category summary", summary)
	}
}

func TestDevLogSummaryZeroLimitKeepsAllEntriesLikePythonSlice(t *testing.T) {
	store := builtin.NewDevLogStore("session-1", "Agent")
	store.Append(builtin.NewDevLogEntry("issue", "first", nil))
	store.Append(builtin.NewDevLogEntry("issue", "second", nil))
	store.Append(builtin.NewDevLogEntry("issue", "third", nil))
	store.Append(builtin.NewDevLogEntry("issue", "fourth", nil))

	summary := store.GenerateSummary(0)
	if !strings.Contains(summary, "最近:") {
		t.Fatalf("summary should include recent entries when limit=0, got %q", summary)
	}
}

func TestDevLogRunReturnsInternalErrorWhenPersistFails(t *testing.T) {
	root := t.TempDir()
	tool := builtin.NewDevLogTool("session-1", "Agent", root, "memory/devlogs")

	blockedPath := filepath.Join(root, "blocked")
	if err := os.WriteFile(blockedPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocked file error: %v", err)
	}
	tool.ExportSetPersistenceDir(blockedPath)

	resp := tool.Run(map[string]any{
		"action":   "append",
		"category": "decision",
		"content":  "persist me",
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInternalError {
		t.Fatalf("error = %#v, want INTERNAL_ERROR", resp.ErrorInfo)
	}
	if !strings.HasPrefix(resp.Text, "日志持久化失败：") {
		t.Fatalf("text = %q, want persist failure prefix", resp.Text)
	}
}

func TestDevLogKeepsEmptySessionAndAgentNamesLikePython(t *testing.T) {
	root := t.TempDir()
	tool := builtin.NewDevLogTool("", "", root, "")

	if tool.ExportGetSessionID() != "" {
		t.Fatalf("sessionID = %q, want empty string", tool.ExportGetSessionID())
	}
	if tool.ExportGetAgentName() != "" {
		t.Fatalf("agentName = %q, want empty string", tool.ExportGetAgentName())
	}
	if filepath.Clean(tool.ExportGetPersistenceDir()) != filepath.Clean(root) {
		t.Fatalf("persistenceDir = %q, want %q", tool.ExportGetPersistenceDir(), root)
	}
}

func TestNewDevLogToolDoesNotPanicWhenPersistenceDirCannotBeCreated(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	tool := builtin.NewDevLogTool("s", "a", blocker, "logs")
	if tool == nil {
		t.Fatalf("NewDevLogTool returned nil when persistence dir cannot be created")
	}
}
