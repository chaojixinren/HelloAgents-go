package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"helloagents-go/hello_agents/tools"
)

func TestTodoWriteRejectsMissingContentField(t *testing.T) {
	tool := NewTodoWriteTool(t.TempDir(), "memory/todos")
	resp := tool.Run(map[string]any{
		"todos": []any{
			map[string]any{"status": "pending"},
		},
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("error = %#v, want INVALID_PARAM", resp.ErrorInfo)
	}
	if !strings.Contains(resp.Text, "content 不能为空") {
		t.Fatalf("text = %q, want content validation message", resp.Text)
	}
}

func TestTodoWriteJSONStringObjectStillValidatesAsArray(t *testing.T) {
	tool := NewTodoWriteTool(t.TempDir(), "memory/todos")
	resp := tool.Run(map[string]any{
		"todos": "{}",
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("error = %#v, want INVALID_PARAM", resp.ErrorInfo)
	}
	if resp.Text != "todos 必须是数组" {
		t.Fatalf("text = %q, want %q", resp.Text, "todos 必须是数组")
	}
}

func TestTodoListGetPendingMatchesPythonSliceForZero(t *testing.T) {
	list := TodoList{
		Todos: []TodoItem{
			{Content: "a", Status: "pending"},
			{Content: "b", Status: "pending"},
		},
	}

	got := list.GetPending(0)
	if len(got) != 0 {
		t.Fatalf("len(GetPending(0)) = %d, want 0", len(got))
	}
}

func TestTodoWriteLoadTodosBackfillsUpdatedAt(t *testing.T) {
	root := t.TempDir()
	tool := NewTodoWriteTool(root, "memory/todos")

	path := filepath.Join(root, "todo.json")
	payload := `{
  "summary": "demo",
  "todos": [
    {"content": "x", "status": "pending", "created_at": "2026-01-01T00:00:00"}
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("seed file write error: %v", err)
	}

	if err := tool.LoadTodos(path); err != nil {
		t.Fatalf("LoadTodos error: %v", err)
	}
	if len(tool.CurrentTodos.Todos) != 1 {
		t.Fatalf("todos length = %d, want 1", len(tool.CurrentTodos.Todos))
	}
	item := tool.CurrentTodos.Todos[0]
	if item.UpdatedAt != item.CreatedAt {
		t.Fatalf("updated_at = %q, want fallback created_at %q", item.UpdatedAt, item.CreatedAt)
	}
}

func TestTodoWriteReturnsInternalErrorWhenPersistFails(t *testing.T) {
	root := t.TempDir()
	tool := NewTodoWriteTool(root, "memory/todos")

	blockedPath := filepath.Join(root, "blocked")
	if err := os.WriteFile(blockedPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocked file error: %v", err)
	}
	tool.PersistenceDir = blockedPath

	resp := tool.Run(map[string]any{
		"todos": []any{
			map[string]any{"content": "task", "status": "pending"},
		},
	})

	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInternalError {
		t.Fatalf("error = %#v, want INTERNAL_ERROR", resp.ErrorInfo)
	}
	if !strings.HasPrefix(resp.Text, "处理任务列表失败：") {
		t.Fatalf("text = %q, want internal failure prefix", resp.Text)
	}
}

func TestTodoWriteEmptyPersistenceDirMapsToProjectRootLikePathJoin(t *testing.T) {
	root := t.TempDir()
	tool := NewTodoWriteTool(root, "")
	if filepath.Clean(tool.PersistenceDir) != filepath.Clean(root) {
		t.Fatalf("PersistenceDir = %q, want %q", tool.PersistenceDir, root)
	}
}

func TestTodoWriteExplicitEmptyActionDoesNotFallbackToCreate(t *testing.T) {
	tool := NewTodoWriteTool(t.TempDir(), "memory/todos")
	resp := tool.Run(map[string]any{
		"action": "",
	})
	if resp.Status != tools.ToolStatusError {
		t.Fatalf("status = %q, want %q", resp.Status, tools.ToolStatusError)
	}
	if resp.ErrorInfo == nil || resp.ErrorInfo["code"] != tools.ToolErrorCodeInvalidParam {
		t.Fatalf("error = %#v, want INVALID_PARAM", resp.ErrorInfo)
	}
	if !strings.Contains(resp.Text, "未知 action") {
		t.Fatalf("text = %q, want unknown action error", resp.Text)
	}
}
