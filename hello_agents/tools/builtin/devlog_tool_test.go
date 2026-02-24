package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"helloagents-go/hello_agents/tools"
)

func TestDevLogAppendAllowsWhitespaceContentLikePython(t *testing.T) {
	tool := NewDevLogTool("session-1", "Agent", t.TempDir(), "memory/devlogs")
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
	store := NewDevLogStore("session-1", "Agent")
	store.Append(NewDevLogEntry("issue", "first", nil))
	store.Append(NewDevLogEntry("decision", "second", nil))

	summary := store.GenerateSummary(10)
	if !strings.Contains(summary, "分类: issue(1), decision(1)") {
		t.Fatalf("summary = %q, want insertion-order category summary", summary)
	}
}

func TestDevLogSummaryZeroLimitKeepsAllEntriesLikePythonSlice(t *testing.T) {
	store := NewDevLogStore("session-1", "Agent")
	store.Append(NewDevLogEntry("issue", "first", nil))
	store.Append(NewDevLogEntry("issue", "second", nil))
	store.Append(NewDevLogEntry("issue", "third", nil))
	store.Append(NewDevLogEntry("issue", "fourth", nil))

	summary := store.GenerateSummary(0)
	if !strings.Contains(summary, "最近:") {
		t.Fatalf("summary should include recent entries when limit=0, got %q", summary)
	}
}

func TestDevLogRunReturnsInternalErrorWhenPersistFails(t *testing.T) {
	root := t.TempDir()
	tool := NewDevLogTool("session-1", "Agent", root, "memory/devlogs")

	blockedPath := filepath.Join(root, "blocked")
	if err := os.WriteFile(blockedPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocked file error: %v", err)
	}
	tool.persistenceDir = blockedPath

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
	if !strings.HasPrefix(resp.Text, "DevLog 操作失败：") {
		t.Fatalf("text = %q, want internal failure prefix", resp.Text)
	}
}

func TestDevLogKeepsEmptySessionAndAgentNamesLikePython(t *testing.T) {
	root := t.TempDir()
	tool := NewDevLogTool("", "", root, "")

	if tool.sessionID != "" {
		t.Fatalf("sessionID = %q, want empty string", tool.sessionID)
	}
	if tool.agentName != "" {
		t.Fatalf("agentName = %q, want empty string", tool.agentName)
	}
	if filepath.Clean(tool.persistenceDir) != filepath.Clean(root) {
		t.Fatalf("persistenceDir = %q, want %q", tool.persistenceDir, root)
	}
}
