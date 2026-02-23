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

var CATEGORIES = map[string]string{
	"decision":    "架构/技术选型决策",
	"progress":    "阶段性进展记录",
	"issue":       "遇到的问题",
	"solution":    "问题解决方案",
	"refactor":    "重构决策",
	"test":        "测试相关记录",
	"performance": "性能优化记录",
}

type DevLogEntry struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Category  string         `json:"category"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
}

func NewDevLogEntry(category, content string, metadata map[string]any) DevLogEntry {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return DevLogEntry{
		ID:        fmt.Sprintf("log-%x", time.Now().UnixNano())[:12],
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Category:  category,
		Content:   content,
		Metadata:  metadata,
	}
}

func (e DevLogEntry) ToMap() map[string]any {
	return map[string]any{
		"id":        e.ID,
		"timestamp": e.Timestamp,
		"category":  e.Category,
		"content":   e.Content,
		"metadata":  e.Metadata,
	}
}

// ToDict keeps naming parity with Python DevLogEntry.to_dict().
func (e DevLogEntry) ToDict() map[string]any {
	return e.ToMap()
}

func DevLogEntryFromMap(data map[string]any) DevLogEntry {
	id, _ := data["id"].(string)
	timestamp, _ := data["timestamp"].(string)
	category, _ := data["category"].(string)
	content, _ := data["content"].(string)
	metadata, _ := data["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	return DevLogEntry{ID: id, Timestamp: timestamp, Category: category, Content: content, Metadata: metadata}
}

type DevLogStore struct {
	SessionID string        `json:"session_id"`
	AgentName string        `json:"agent_name"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
	Entries   []DevLogEntry `json:"entries"`
}

func NewDevLogStore(sessionID, agentName string) DevLogStore {
	now := time.Now().Format(time.RFC3339Nano)
	return DevLogStore{
		SessionID: sessionID,
		AgentName: agentName,
		CreatedAt: now,
		UpdatedAt: now,
		Entries:   []DevLogEntry{},
	}
}

func (s *DevLogStore) Append(entry DevLogEntry) {
	s.Entries = append(s.Entries, entry)
	s.UpdatedAt = time.Now().Format(time.RFC3339Nano)
}

func (s *DevLogStore) FilterEntries(category string, tags []string, limit int) []DevLogEntry {
	filtered := make([]DevLogEntry, 0)
	for _, e := range s.Entries {
		if category != "" && e.Category != category {
			continue
		}
		if len(tags) > 0 {
			rawTags, _ := e.Metadata["tags"].([]any)
			match := false
			for _, tag := range tags {
				for _, rt := range rawTags {
					if fmt.Sprintf("%v", rt) == tag {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, e)
	}
	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

func (s *DevLogStore) GetStats() map[string]any {
	byCategory := map[string]int{}
	for _, e := range s.Entries {
		byCategory[e.Category]++
	}
	return map[string]any{
		"total_entries": len(s.Entries),
		"by_category":   byCategory,
	}
}

func (s *DevLogStore) GenerateSummary(limit int) string {
	if len(s.Entries) == 0 {
		return "📝 暂无开发日志"
	}
	stats := s.GetStats()
	byCategory, _ := stats["by_category"].(map[string]int)
	catParts := make([]string, 0, len(byCategory))
	for cat, count := range byCategory {
		catParts = append(catParts, fmt.Sprintf("%s(%d)", cat, count))
	}

	recent := s.Entries
	if limit > 0 && len(recent) > limit {
		recent = recent[len(recent)-limit:]
	}
	recentParts := make([]string, 0)
	for _, e := range recent {
		content := e.Content
		if len(content) > 30 {
			content = content[:30] + "..."
		}
		recentParts = append(recentParts, fmt.Sprintf("[%s] %s", e.Category, content))
		if len(recentParts) >= 3 {
			break
		}
	}

	return fmt.Sprintf("📝 共 %d 条日志. 分类: %s. 最近: %s",
		stats["total_entries"].(int),
		strings.Join(catParts, ", "),
		strings.Join(recentParts, "; "))
}

func (s *DevLogStore) ToMap() map[string]any {
	entries := make([]map[string]any, 0, len(s.Entries))
	for _, e := range s.Entries {
		entries = append(entries, e.ToMap())
	}
	return map[string]any{
		"session_id": s.SessionID,
		"agent_name": s.AgentName,
		"created_at": s.CreatedAt,
		"updated_at": s.UpdatedAt,
		"entries":    entries,
		"stats":      s.GetStats(),
	}
}

// ToDict keeps naming parity with Python DevLogStore.to_dict().
func (s *DevLogStore) ToDict() map[string]any {
	return s.ToMap()
}

func DevLogStoreFromMap(data map[string]any) DevLogStore {
	sessionID, _ := data["session_id"].(string)
	agentName, _ := data["agent_name"].(string)
	createdAt, _ := data["created_at"].(string)
	updatedAt, _ := data["updated_at"].(string)
	store := DevLogStore{SessionID: sessionID, AgentName: agentName, CreatedAt: createdAt, UpdatedAt: updatedAt, Entries: []DevLogEntry{}}
	if rawEntries, ok := data["entries"].([]any); ok {
		for _, raw := range rawEntries {
			if m, ok := raw.(map[string]any); ok {
				store.Entries = append(store.Entries, DevLogEntryFromMap(m))
			}
		}
	}
	return store
}

type DevLogTool struct {
	tools.BaseTool
	SessionID      string
	AgentName      string
	ProjectRoot    string
	PersistenceDir string
	Store          DevLogStore
}

func NewDevLogTool(sessionID, agentName, projectRoot, persistenceDir string) *DevLogTool {
	if sessionID == "" {
		sessionID = fmt.Sprintf("s-%d", time.Now().Unix())
	}
	if agentName == "" {
		agentName = "Agent"
	}
	if projectRoot == "" {
		projectRoot = "."
	}
	if persistenceDir == "" {
		persistenceDir = "memory/devlogs"
	}
	base := tools.NewBaseTool("DevLog", "记录开发过程中的关键决策和问题", false)
	base.Parameters = map[string]tools.ToolParameter{
		"action": {
			Name:        "action",
			Type:        "string",
			Description: "append/read/summary/clear",
			Required:    true,
		},
		"category": {
			Name:        "category",
			Type:        "string",
			Description: "日志类别",
			Required:    false,
		},
		"content": {
			Name:        "content",
			Type:        "string",
			Description: "日志内容",
			Required:    false,
		},
		"metadata": {
			Name:        "metadata",
			Type:        "object",
			Description: "扩展元数据",
			Required:    false,
		},
		"tags": {
			Name:        "tags",
			Type:        "array",
			Description: "标签过滤（read）",
			Required:    false,
		},
		"limit": {
			Name:        "limit",
			Type:        "integer",
			Description: "读取条数限制",
			Required:    false,
			Default:     20,
		},
	}
	fullDir := filepath.Join(projectRoot, persistenceDir)
	_ = os.MkdirAll(fullDir, 0o755)
	tool := &DevLogTool{
		BaseTool:       base,
		SessionID:      sessionID,
		AgentName:      agentName,
		ProjectRoot:    projectRoot,
		PersistenceDir: fullDir,
		Store:          NewDevLogStore(sessionID, agentName),
	}
	tool.loadIfExists()
	return tool
}

func (t *DevLogTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *DevLogTool) Run(parameters map[string]any) tools.ToolResponse {
	action, _ := parameters["action"].(string)
	if action == "" {
		return tools.Error("缺少 action 参数", tools.ToolErrorCodeInvalidParam, nil)
	}
	switch action {
	case "append":
		return t.handleAppend(parameters)
	case "read":
		return t.handleRead(parameters)
	case "summary":
		return t.handleSummary()
	case "clear":
		return t.handleClear()
	default:
		return tools.Error("不支持的 action", tools.ToolErrorCodeInvalidParam, map[string]any{"action": action})
	}
}

func (t *DevLogTool) handleAppend(parameters map[string]any) tools.ToolResponse {
	category, _ := parameters["category"].(string)
	content, _ := parameters["content"].(string)
	metadata, _ := parameters["metadata"].(map[string]any)
	if category == "" || content == "" {
		return tools.Error("append 需要 category 和 content", tools.ToolErrorCodeInvalidParam, nil)
	}
	if _, ok := CATEGORIES[category]; !ok {
		return tools.Error("无效 category", tools.ToolErrorCodeInvalidParam, map[string]any{"category": category})
	}
	entry := NewDevLogEntry(category, content, metadata)
	t.Store.Append(entry)
	t.persist()
	return tools.Success("日志已记录", map[string]any{"entry": entry.ToMap(), "stats": t.Store.GetStats()}, nil)
}

func (t *DevLogTool) handleRead(parameters map[string]any) tools.ToolResponse {
	category, _ := parameters["category"].(string)
	limit := 20
	if v, ok := parameters["limit"].(int); ok && v > 0 {
		limit = v
	}
	tags := []string{}
	if rawTags, ok := parameters["tags"].([]any); ok {
		for _, raw := range rawTags {
			tags = append(tags, fmt.Sprintf("%v", raw))
		}
	}
	entries := t.Store.FilterEntries(category, tags, limit)
	lines := make([]string, 0, len(entries))
	entryMaps := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("[%s] %s - %s", e.Timestamp, e.Category, e.Content))
		entryMaps = append(entryMaps, e.ToMap())
	}
	if len(lines) == 0 {
		return tools.Success("暂无匹配日志", map[string]any{"entries": entryMaps}, nil)
	}
	return tools.Success(strings.Join(lines, "\n"), map[string]any{"entries": entryMaps, "count": len(entries)}, nil)
}

func (t *DevLogTool) handleSummary() tools.ToolResponse {
	summary := t.Store.GenerateSummary(10)
	return tools.Success(summary, map[string]any{"stats": t.Store.GetStats()}, nil)
}

func (t *DevLogTool) handleClear() tools.ToolResponse {
	t.Store = NewDevLogStore(t.SessionID, t.AgentName)
	t.persist()
	return tools.Success("开发日志已清空", map[string]any{"stats": t.Store.GetStats()}, nil)
}

func (t *DevLogTool) persist() {
	path := filepath.Join(t.PersistenceDir, fmt.Sprintf("devlog-%s-%s.json", t.SessionID, t.AgentName))
	payload, _ := json.MarshalIndent(t.Store.ToMap(), "", "  ")
	tmpPath := path + ".tmp"
	_ = os.WriteFile(tmpPath, payload, 0o644)
	_ = os.Rename(tmpPath, path)
}

func (t *DevLogTool) loadIfExists() {
	path := filepath.Join(t.PersistenceDir, fmt.Sprintf("devlog-%s-%s.json", t.SessionID, t.AgentName))
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	t.Store = DevLogStoreFromMap(raw)
}
