package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

type TodoItem struct {
	Content   string `json:"content"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TodoList struct {
	Summary string     `json:"summary"`
	Todos   []TodoItem `json:"todos"`
}

func (l *TodoList) GetInProgress() *TodoItem {
	for i := range l.Todos {
		if l.Todos[i].Status == "in_progress" {
			return &l.Todos[i]
		}
	}
	return nil
}

func (l *TodoList) GetPending(limit int) []TodoItem {
	out := make([]TodoItem, 0)
	for _, item := range l.Todos {
		if item.Status == "pending" {
			out = append(out, item)
		}
	}

	// Match Python slicing behavior: pending[:limit]
	if limit >= 0 {
		if limit > len(out) {
			limit = len(out)
		}
		return out[:limit]
	}

	end := len(out) + limit
	if end < 0 {
		end = 0
	}
	return out[:end]
}

func (l *TodoList) GetCompleted() []TodoItem {
	out := make([]TodoItem, 0)
	for _, item := range l.Todos {
		if item.Status == "completed" {
			out = append(out, item)
		}
	}
	return out
}

func (l *TodoList) GetStats() map[string]int {
	total := len(l.Todos)
	completed := 0
	inProgress := 0
	for _, t := range l.Todos {
		if t.Status == "completed" {
			completed++
		}
		if t.Status == "in_progress" {
			inProgress++
		}
	}
	pending := total - completed - inProgress
	if pending < 0 {
		pending = 0
	}
	return map[string]int{
		"total":       total,
		"completed":   completed,
		"in_progress": inProgress,
		"pending":     pending,
	}
}

type TodoWriteTool struct {
	tools.BaseTool
	ProjectRoot    string
	PersistenceDir string
	CurrentTodos   TodoList
}

func NewTodoWriteTool(projectRoot string, persistenceDir string) *TodoWriteTool {
	if projectRoot == "" {
		projectRoot = "."
	}
	base := tools.NewBaseTool("TodoWrite", "管理任务列表，保持单线程专注", false)
	base.Parameters = map[string]tools.ToolParameter{
		"summary": {
			Name:        "summary",
			Type:        "string",
			Description: "总体任务描述",
			Required:    false,
			Default:     "",
		},
		"todos": {
			Name:        "todos",
			Type:        "array",
			Description: "任务数组 [{content,status}]",
			Required:    false,
			Default:     []any{},
		},
		"action": {
			Name:        "action",
			Type:        "string",
			Description: "create/update/clear",
			Required:    false,
			Default:     "create",
		},
	}
	fullDir := filepath.Join(projectRoot, persistenceDir)
	_ = os.MkdirAll(fullDir, 0o755)
	return &TodoWriteTool{
		BaseTool:       base,
		ProjectRoot:    projectRoot,
		PersistenceDir: fullDir,
		CurrentTodos:   TodoList{Summary: "", Todos: []TodoItem{}},
	}
}

func (t *TodoWriteTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *TodoWriteTool) Run(parameters map[string]any) (resp tools.ToolResponse) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resp = tools.Error(
				fmt.Sprintf("处理任务列表失败：%v", recovered),
				tools.ToolErrorCodeInternalError,
				nil,
			)
		}
	}()

	action := "create"
	if rawAction, exists := parameters["action"]; exists {
		action, _ = rawAction.(string)
	}
	if action == "clear" {
		t.CurrentTodos = TodoList{Summary: "", Todos: []TodoItem{}}
		return tools.Success("✅ 任务列表已清空", map[string]any{
			"action":  action,
			"summary": "",
			"stats": map[string]int{
				"total":       0,
				"completed":   0,
				"in_progress": 0,
				"pending":     0,
			},
		}, nil)
	}
	if action != "create" && action != "update" {
		return tools.Error(fmt.Sprintf("未知 action: %s", action), tools.ToolErrorCodeInvalidParam, nil)
	}

	todosData := parameters["todos"]
	if todosData == nil {
		todosData = []any{}
	}

	if todosJSON, ok := todosData.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(todosJSON), &parsed); err != nil {
			return tools.Error(fmt.Sprintf("todos JSON 格式错误：%v", err), tools.ToolErrorCodeInvalidParam, nil)
		}
		todosData = parsed
	}

	validation := t.validateTodos(todosData)
	if !validation["valid"].(bool) {
		return tools.Error(validation["message"].(string), tools.ToolErrorCodeInvalidParam, nil)
	}
	rawList := todosData.([]any)

	now := nowPythonISOTime()
	items := make([]TodoItem, 0, len(rawList))
	for _, raw := range rawList {
		obj, _ := raw.(map[string]any)
		createdAt := now
		if rawCreatedAt, exists := obj["created_at"]; exists {
			createdAt = todoStringValue(rawCreatedAt)
		}

		item := TodoItem{
			Content:   todoStringValue(obj["content"]),
			Status:    todoStringValue(obj["status"]),
			CreatedAt: createdAt,
			UpdatedAt: now,
		}
		items = append(items, item)
	}

	summary := ""
	if rawSummary, exists := parameters["summary"]; exists {
		summary = todoStringValue(rawSummary)
	}
	t.CurrentTodos = TodoList{Summary: summary, Todos: items}
	recap := t.generateRecap()
	if err := t.persistTodos(); err != nil {
		return tools.Error(
			fmt.Sprintf("任务列表持久化失败：%v", err),
			tools.ToolErrorCodeInternalError,
			nil,
		)
	}

	return tools.Success(recap, map[string]any{
		"action":  action,
		"summary": t.CurrentTodos.Summary,
		"stats":   t.CurrentTodos.GetStats(),
	}, nil)
}

func (t *TodoWriteTool) validateTodos(todosData any) map[string]any {
	rawList, ok := todosData.([]any)
	if !ok {
		return map[string]any{"valid": false, "message": "todos 必须是数组"}
	}

	inProgressCount := 0
	for i, raw := range rawList {
		obj, ok := raw.(map[string]any)
		if !ok {
			return map[string]any{"valid": false, "message": fmt.Sprintf("第 %d 个任务必须是对象", i+1)}
		}
		content := strings.TrimSpace(todoStringValue(obj["content"]))
		status := todoStringValue(obj["status"])
		if content == "" {
			return map[string]any{"valid": false, "message": fmt.Sprintf("第 %d 个任务的 content 不能为空", i+1)}
		}
		if status != "pending" && status != "in_progress" && status != "completed" {
			return map[string]any{"valid": false, "message": fmt.Sprintf("第 %d 个任务的 status 必须是 pending/in_progress/completed", i+1)}
		}
		if status == "in_progress" {
			inProgressCount++
		}
	}
	if inProgressCount > 1 {
		return map[string]any{"valid": false, "message": fmt.Sprintf("最多只能有 1 个 in_progress 任务，当前有 %d 个", inProgressCount)}
	}
	return map[string]any{"valid": true, "message": ""}
}

func (t *TodoWriteTool) generateRecap() string {
	stats := t.CurrentTodos.GetStats()
	if stats["total"] == 0 {
		return "📋 [0/0] 无活动任务"
	}
	if stats["completed"] == stats["total"] && stats["total"] > 0 {
		return fmt.Sprintf("✅ [%d/%d] 所有任务已完成！", stats["completed"], stats["total"])
	}

	parts := []string{fmt.Sprintf("📋 [%d/%d]", stats["completed"], stats["total"])}
	if current := t.CurrentTodos.GetInProgress(); current != nil {
		parts = append(parts, "进行中: "+current.Content)
	}
	pending := t.CurrentTodos.GetPending(3)
	if len(pending) > 0 {
		names := make([]string, 0, len(pending))
		for _, p := range pending {
			names = append(names, p.Content)
		}
		parts = append(parts, "待处理: "+strings.Join(names, "; "))
	}
	if stats["pending"] > 3 {
		parts = append(parts, fmt.Sprintf("还有 %d 个...", stats["pending"]-3))
	}
	return strings.Join(parts, ". ")
}

func (t *TodoWriteTool) persistTodos() error {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("todoList-%s.json", timestamp)
	filePath := filepath.Join(t.PersistenceDir, filename)

	data := map[string]any{
		"summary":    t.CurrentTodos.Summary,
		"todos":      t.CurrentTodos.Todos,
		"created_at": nowPythonISOTime(),
		"stats":      t.CurrentTodos.GetStats(),
	}

	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return err
	}
	return nil
}

func (t *TodoWriteTool) LoadTodos(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	var raw struct {
		Summary string     `json:"summary"`
		Todos   []TodoItem `json:"todos"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for idx := range raw.Todos {
		if raw.Todos[idx].UpdatedAt == "" {
			raw.Todos[idx].UpdatedAt = raw.Todos[idx].CreatedAt
		}
	}
	t.CurrentTodos = TodoList{Summary: raw.Summary, Todos: raw.Todos}
	return nil
}

func todoStringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}
