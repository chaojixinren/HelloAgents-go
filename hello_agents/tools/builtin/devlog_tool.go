package builtin

import (
	"crypto/rand"
	"encoding/hex"
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

var categoryOrder = []string{
	"decision",
	"progress",
	"issue",
	"solution",
	"refactor",
	"test",
	"performance",
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
		ID:        "log-" + randomHex(4),
		Timestamp: nowPythonISOTime(),
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
	return DevLogEntry{
		ID:        id,
		Timestamp: timestamp,
		Category:  category,
		Content:   content,
		Metadata:  metadata,
	}
}

type DevLogStore struct {
	SessionID string        `json:"session_id"`
	AgentName string        `json:"agent_name"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
	Entries   []DevLogEntry `json:"entries"`
}

func NewDevLogStore(sessionID, agentName string) DevLogStore {
	now := nowPythonISOTime()
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
	s.UpdatedAt = nowPythonISOTime()
}

func (s *DevLogStore) FilterEntries(category string, tags []string, limit int) []DevLogEntry {
	filtered := make([]DevLogEntry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if category != "" && entry.Category != category {
			continue
		}
		if len(tags) > 0 && !entryContainsAnyTag(entry, tags) {
			continue
		}
		filtered = append(filtered, entry)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

func (s *DevLogStore) GetStats() map[string]any {
	byCategory := map[string]int{}
	for _, entry := range s.Entries {
		byCategory[entry.Category]++
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
	total, _ := stats["total_entries"].(int)
	byCategory, _ := stats["by_category"].(map[string]int)

	summaryParts := []string{fmt.Sprintf("📝 共 %d 条日志", total)}

	// Follow Python dict insertion semantics: category order is first appearance in entries.
	categoryParts := make([]string, 0, len(byCategory))
	seenCategories := map[string]struct{}{}
	for _, entry := range s.Entries {
		if _, seen := seenCategories[entry.Category]; seen {
			continue
		}
		seenCategories[entry.Category] = struct{}{}
		categoryParts = append(categoryParts, fmt.Sprintf("%s(%d)", entry.Category, byCategory[entry.Category]))
	}
	summaryParts = append(summaryParts, "分类: "+strings.Join(categoryParts, ", "))

	recent := s.Entries
	switch {
	case limit > 0:
		if len(recent) > limit {
			recent = recent[len(recent)-limit:]
		}
	case limit == 0:
		// Python slice behavior: entries[-0:] keeps all entries.
	default:
		start := -limit
		if start > len(recent) {
			start = len(recent)
		}
		recent = recent[start:]
	}
	if len(recent) > 0 {
		start := len(recent) - 3
		if start < 0 {
			start = 0
		}
		recentParts := make([]string, 0, len(recent)-start)
		for _, entry := range recent[start:] {
			content := entry.Content
			if len(content) > 30 {
				content = content[:30] + "..."
			}
			recentParts = append(recentParts, fmt.Sprintf("[%s] %s", entry.Category, content))
		}
		summaryParts = append(summaryParts, "最近: "+strings.Join(recentParts, "; "))
	}

	return strings.Join(summaryParts, ". ")
}

func (s *DevLogStore) ToMap() map[string]any {
	entries := make([]map[string]any, 0, len(s.Entries))
	for _, entry := range s.Entries {
		entries = append(entries, entry.ToMap())
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
	entries := []DevLogEntry{}

	if rawEntries, ok := data["entries"].([]any); ok {
		for _, raw := range rawEntries {
			entryData, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			entries = append(entries, DevLogEntryFromMap(entryData))
		}
	}

	return DevLogStore{
		SessionID: sessionID,
		AgentName: agentName,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Entries:   entries,
	}
}

type DevLogTool struct {
	tools.BaseTool
	sessionID      string
	agentName      string
	projectRoot    string
	persistenceDir string
	store          DevLogStore
}

func NewDevLogTool(sessionID, agentName, projectRoot, persistenceDir string) *DevLogTool {
	if projectRoot == "" {
		projectRoot = "."
	}

	categoryLines := make([]string, 0, len(categoryOrder))
	for _, category := range categoryOrder {
		categoryLines = append(categoryLines, fmt.Sprintf("- %s: %s", category, CATEGORIES[category]))
	}

	base := tools.NewBaseTool(
		"DevLog",
		fmt.Sprintf(`记录开发过程中的关键决策和问题。

支持的类别：
%s

操作：
- append: 追加日志（需要 category, content, metadata）
- read: 读取日志（可选 category, tags, limit）
- summary: 生成摘要
- clear: 清空日志

示例：
{
  "action": "append",
  "category": "decision",
  "content": "选择使用 Redis 作为缓存层",
  "metadata": {"tags": ["architecture", "cache"]}
}`, strings.Join(categoryLines, "\n")),
		false,
	)
	base.Parameters = map[string]tools.ToolParameter{
		"action": {
			Name:        "action",
			Type:        "string",
			Description: "操作类型：append（追加）、read（读取）、summary（摘要）、clear（清空）",
			Required:    true,
		},
		"category": {
			Name:        "category",
			Type:        "string",
			Description: fmt.Sprintf("日志类别（append 时必填）：%s", strings.Join(categoryOrder, ", ")),
			Required:    false,
		},
		"content": {
			Name:        "content",
			Type:        "string",
			Description: "日志内容（append 时必填）",
			Required:    false,
		},
		"metadata": {
			Name:        "metadata",
			Type:        "object",
			Description: `元数据（可选），如 {"tags": ["cache"], "step": 3, "related_tool": "WriteTool"}`,
			Required:    false,
		},
		"filter": {
			Name:        "filter",
			Type:        "object",
			Description: `过滤条件（read 时可选），如 {"category": "decision", "tags": ["architecture"], "limit": 10}`,
			Required:    false,
		},
	}

	fullPersistenceDir := filepath.Join(projectRoot, persistenceDir)
	if err := os.MkdirAll(fullPersistenceDir, 0o755); err != nil {
		panic(err)
	}

	tool := &DevLogTool{
		BaseTool:       base,
		sessionID:      sessionID,
		agentName:      agentName,
		projectRoot:    projectRoot,
		persistenceDir: fullPersistenceDir,
		store:          NewDevLogStore(sessionID, agentName),
	}
	tool.loadIfExists()
	return tool
}

func (t *DevLogTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *DevLogTool) Run(parameters map[string]any) (resp tools.ToolResponse) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resp = tools.Error(
				fmt.Sprintf("DevLog 操作失败：%v", recovered),
				tools.ToolErrorCodeInternalError,
				nil,
			)
		}
	}()

	action, _ := parameters["action"].(string)

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
		return tools.Error(
			fmt.Sprintf("未知操作：%v", parameters["action"]),
			tools.ToolErrorCodeInvalidParam,
			nil,
		)
	}
}

func (t *DevLogTool) handleAppend(parameters map[string]any) tools.ToolResponse {
	category := devlogStringValue(parameters["category"])
	content := devlogStringValue(parameters["content"])

	if category == "" {
		return tools.Error("追加日志时必须指定 category", tools.ToolErrorCodeInvalidParam, nil)
	}
	if _, ok := CATEGORIES[category]; !ok {
		return tools.Error(
			fmt.Sprintf("无效的类别：%s，支持的类别：%s", category, strings.Join(categoryOrder, ", ")),
			tools.ToolErrorCodeInvalidParam,
			nil,
		)
	}
	if content == "" {
		return tools.Error("追加日志时必须指定 content", tools.ToolErrorCodeInvalidParam, nil)
	}

	metadata, _ := parameters["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}

	entry := NewDevLogEntry(category, content, metadata)
	t.store.Append(entry)
	if err := t.persist(); err != nil {
		panic(err)
	}

	displayContent := content
	if len(displayContent) > 50 {
		displayContent = displayContent[:50] + "..."
	}

	return tools.Success(
		fmt.Sprintf("✅ 日志已记录 [%s]: %s", category, displayContent),
		map[string]any{
			"log_id":    entry.ID,
			"timestamp": entry.Timestamp,
			"category":  entry.Category,
		},
		t.store.GetStats(),
	)
}

func (t *DevLogTool) handleRead(parameters map[string]any) tools.ToolResponse {
	filterParams := map[string]any{}
	if rawFilter, ok := parameters["filter"].(map[string]any); ok && rawFilter != nil {
		filterParams = rawFilter
	}

	category, _ := filterParams["category"].(string)
	tags := parseStringList(filterParams["tags"])
	limit := intFromAny(filterParams["limit"])

	entries := t.store.FilterEntries(category, tags, limit)
	if len(entries) == 0 {
		return tools.Success(
			"📝 未找到匹配的日志",
			map[string]any{"entries": []map[string]any{}},
			map[string]any{"matched": 0},
		)
	}

	lines := []string{fmt.Sprintf("📝 找到 %d 条日志：\n", len(entries))}
	serializedEntries := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("[%s] %s", entry.Category, entry.Timestamp))
		lines = append(lines, "  "+entry.Content)
		if len(entry.Metadata) > 0 {
			metadataBytes, _ := json.Marshal(entry.Metadata)
			lines = append(lines, "  元数据: "+string(metadataBytes))
		}
		lines = append(lines, "")
		serializedEntries = append(serializedEntries, entry.ToMap())
	}

	return tools.Success(
		strings.Join(lines, "\n"),
		map[string]any{"entries": serializedEntries},
		map[string]any{"matched": len(entries)},
	)
}

func (t *DevLogTool) handleSummary() tools.ToolResponse {
	summary := t.store.GenerateSummary(10)
	return tools.Success(summary, t.store.GetStats())
}

func (t *DevLogTool) handleClear() tools.ToolResponse {
	oldCount := len(t.store.Entries)
	t.store.Entries = []DevLogEntry{}
	t.store.UpdatedAt = nowPythonISOTime()
	if err := t.persist(); err != nil {
		panic(err)
	}

	return tools.Success(
		fmt.Sprintf("✅ 已清空 %d 条日志", oldCount),
		map[string]any{"cleared_count": oldCount},
	)
}

func (t *DevLogTool) persist() error {
	filename := fmt.Sprintf("devlog-%s.json", t.sessionID)
	filePath := filepath.Join(t.persistenceDir, filename)

	payload, err := json.MarshalIndent(t.store.ToMap(), "", "  ")
	if err != nil {
		return err
	}

	tmpPath := strings.TrimSuffix(filePath, ".json") + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return err
	}
	return nil
}

func (t *DevLogTool) loadIfExists() {
	filename := fmt.Sprintf("devlog-%s.json", t.sessionID)
	filepath := filepath.Join(t.persistenceDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return
	}

	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	t.store = DevLogStoreFromMap(raw)
}

func entryContainsAnyTag(entry DevLogEntry, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	rawTags, ok := entry.Metadata["tags"]
	if !ok {
		return false
	}
	entryTags := parseStringList(rawTags)
	if len(entryTags) == 0 {
		return false
	}

	for _, wanted := range tags {
		for _, existing := range entryTags {
			if wanted == existing {
				return true
			}
		}
	}
	return false
}

func parseStringList(value any) []string {
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := devlogStringValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := devlogStringValue(typed)
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func randomHex(size int) string {
	if size <= 0 {
		size = 4
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano())[:size*2]
	}
	return hex.EncodeToString(b)
}

func devlogStringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}
