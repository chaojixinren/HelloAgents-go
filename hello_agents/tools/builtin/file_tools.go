package builtin

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"helloagents-go/hello_agents/tools"
)

type ReadTool struct {
	tools.BaseTool
	ProjectRoot string
	WorkingDir  string
	Registry    *tools.ToolRegistry
}

func NewReadTool(projectRoot string, registry *tools.ToolRegistry) *ReadTool {
	return NewReadToolWithOptions(projectRoot, "", registry)
}

func NewReadToolWithOptions(projectRoot string, workingDir string, registry *tools.ToolRegistry) *ReadTool {
	if projectRoot == "" {
		projectRoot = "."
	}
	absRoot, _ := filepath.Abs(projectRoot)
	absWorkingDir := absRoot
	if workingDir != "" {
		absWorkingDir, _ = filepath.Abs(workingDir)
	}
	base := tools.NewBaseTool("Read", "读取文件内容或列出目录内容，支持行号范围和元数据缓存", false)
	base.Parameters = map[string]tools.ToolParameter{
		"path": {
			Name:        "path",
			Type:        "string",
			Description: "要读取的文件路径或目录路径（相对项目根目录）。如果是目录，将列出目录内容",
			Required:    true,
		},
		"offset": {
			Name:        "offset",
			Type:        "integer",
			Description: "起始行号（从 0 开始，仅读取文件时有效）",
			Required:    false,
			Default:     0,
		},
		"limit": {
			Name:        "limit",
			Type:        "integer",
			Description: "最大行数（仅读取文件时有效）",
			Required:    false,
			Default:     2000,
		},
	}
	t := &ReadTool{
		BaseTool:    base,
		ProjectRoot: absRoot,
		WorkingDir:  absWorkingDir,
		Registry:    registry,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *ReadTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *ReadTool) Run(parameters map[string]any) tools.ToolResponse {
	path, _ := parameters["path"].(string)
	offset := intFromAny(parameters["offset"])
	if offset < 0 {
		offset = 0
	}
	rawLimit, hasLimit := parameters["limit"]
	limit := intFromAny(rawLimit)
	if !hasLimit {
		limit = 2000
	}

	if path == "" {
		return tools.Error("缺少必需参数: path", tools.ToolErrorCodeInvalidParam, nil)
	}

	fullPath := t.resolvePath(path)
	st, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tools.Error(fmt.Sprintf("路径 '%s' 不存在", path), tools.ToolErrorCodeNotFound, nil)
		}
		if os.IsPermission(err) {
			return tools.Error(fmt.Sprintf("无权限读取 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("读取文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	if st.IsDir() {
		return t.listDirectory(path, fullPath)
	}

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsPermission(err) {
			return tools.Error(fmt.Sprintf("无权限读取 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("读取文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	allLines := strings.SplitAfter(string(contentBytes), "\n")
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}
	totalLines := len(allLines)
	if offset > len(allLines) {
		offset = len(allLines)
	}
	lines := allLines[offset:]
	if limit > 0 && len(lines) > limit {
		lines = lines[:limit]
	}
	content := strings.Join(lines, "")

	fileMtimeMS := st.ModTime().UnixMilli()
	fileSizeBytes := st.Size()

	if t.Registry != nil {
		t.Registry.CacheReadMetadata(path, map[string]any{
			"file_mtime_ms":   fileMtimeMS,
			"file_size_bytes": fileSizeBytes,
		})
	}

	summary := fmt.Sprintf("读取 %d 行（共 %d 行，%d 字节）\n\n%s", len(lines), totalLines, fileSizeBytes, content)
	return tools.Success(
		summary,
		map[string]any{
			"content":         content,
			"lines":           len(lines),
			"total_lines":     totalLines,
			"file_mtime_ms":   fileMtimeMS,
			"file_size_bytes": fileSizeBytes,
			"offset":          offset,
			"limit":           limit,
		},
	)
}

func (t *ReadTool) listDirectory(path string, fullPath string) tools.ToolResponse {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsPermission(err) {
			return tools.Error(fmt.Sprintf("无权访问目录 '%s'", path), tools.ToolErrorCodeAccessDenied, nil)
		}
		return tools.Error(fmt.Sprintf("列出目录失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	type item struct {
		name     string
		isDir    bool
		sizeStr  string
		mtimeStr string
		relPath  string
	}
	items := make([]item, 0, len(entries))
	totalFiles := 0
	totalDirs := 0

	for _, entry := range entries {
		entryPath := filepath.Join(fullPath, entry.Name())
		st, err := os.Stat(entryPath)
		if err != nil {
			continue
		}
		it := item{name: entry.Name(), isDir: st.IsDir()}
		if st.IsDir() {
			it.sizeStr = "<DIR>"
			totalDirs++
		} else {
			it.sizeStr = formatSize(st.Size())
			totalFiles++
		}
		it.mtimeStr = formatTime(st.ModTime())
		rel, err := filepath.Rel(t.ProjectRoot, entryPath)
		if err != nil {
			rel = entryPath
		}
		it.relPath = filepath.ToSlash(rel)
		items = append(items, it)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir && !items[j].isDir
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})

	if len(items) == 0 {
		return tools.Success(fmt.Sprintf("目录 '%s' 为空", path), map[string]any{
			"path":         path,
			"entries":      []map[string]any{},
			"total_files":  0,
			"total_dirs":   0,
			"is_directory": true,
		})
	}

	lines := []string{fmt.Sprintf("目录 '%s' 包含 %d 个文件，%d 个目录：\n", path, totalFiles, totalDirs)}
	entryMaps := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typeIcon := "📄"
		typeName := "file"
		if item.isDir {
			typeIcon = "📁"
			typeName = "directory"
		}
		lines = append(lines, fmt.Sprintf("%s %-40s %10s %s", typeIcon, item.name, item.sizeStr, item.mtimeStr))
		entryMaps = append(entryMaps, map[string]any{
			"name":  item.name,
			"type":  typeName,
			"size":  item.sizeStr,
			"mtime": item.mtimeStr,
			"path":  item.relPath,
		})
	}

	return tools.Success(strings.Join(lines, "\n"), map[string]any{
		"path":         path,
		"entries":      entryMaps,
		"total_files":  totalFiles,
		"total_dirs":   totalDirs,
		"is_directory": true,
	})
}

func (t *ReadTool) resolvePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(normalized) {
		return normalized
	}
	return filepath.Join(t.WorkingDir, normalized)
}

type WriteTool struct {
	tools.BaseTool
	ProjectRoot string
	WorkingDir  string
	Registry    *tools.ToolRegistry
}

func NewWriteTool(projectRoot string) *WriteTool {
	return NewWriteToolWithOptions(projectRoot, "", nil)
}

func NewWriteToolWithOptions(projectRoot string, workingDir string, registry *tools.ToolRegistry) *WriteTool {
	if projectRoot == "" {
		projectRoot = "."
	}
	absRoot, _ := filepath.Abs(projectRoot)
	absWorkingDir := absRoot
	if workingDir != "" {
		absWorkingDir, _ = filepath.Abs(workingDir)
	}
	base := tools.NewBaseTool("Write", "创建或覆盖文件，支持冲突检测和原子写入", false)
	base.Parameters = map[string]tools.ToolParameter{
		"path":          {Name: "path", Type: "string", Description: "文件路径（相对项目根目录）", Required: true},
		"content":       {Name: "content", Type: "string", Description: "文件内容", Required: true},
		"file_mtime_ms": {Name: "file_mtime_ms", Type: "integer", Description: "缓存的文件修改时间（用于冲突检测）", Required: false},
	}
	t := &WriteTool{
		BaseTool:    base,
		ProjectRoot: absRoot,
		WorkingDir:  absWorkingDir,
		Registry:    registry,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *WriteTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *WriteTool) Run(parameters map[string]any) tools.ToolResponse {
	path, _ := parameters["path"].(string)
	content, contentExists := parameters["content"]
	rawCachedMtime, hasCachedMtime := parameters["file_mtime_ms"]
	cachedMtime := int64(intFromAny(rawCachedMtime))

	if path == "" {
		return tools.Error("缺少必需参数: path", tools.ToolErrorCodeInvalidParam, nil)
	}
	if !contentExists {
		return tools.Error("缺少必需参数: content", tools.ToolErrorCodeInvalidParam, nil)
	}

	contentText := fmt.Sprintf("%v", content)
	fullPath := t.resolvePath(path)
	var backupPath string

	if st, err := os.Stat(fullPath); err == nil {
		currentMtime := st.ModTime().UnixMilli()
		if hasCachedMtime && currentMtime != cachedMtime {
			return tools.Error(
				fmt.Sprintf("文件自上次读取后被修改。当前 mtime=%d, 缓存 mtime=%d", currentMtime, cachedMtime),
				tools.ToolErrorCodeConflict,
				map[string]any{"current_mtime_ms": currentMtime, "cached_mtime_ms": cachedMtime},
			)
		}
		if b, err := t.backupFile(fullPath); err == nil {
			backupPath = b
		} else {
			if isPermissionError(err) {
				return tools.Error(fmt.Sprintf("无权限写入 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
			}
			return tools.Error(fmt.Sprintf("写入文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
		}
	} else {
		if !os.IsNotExist(err) {
			if isPermissionError(err) {
				return tools.Error(fmt.Sprintf("无权限写入 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
			}
			return tools.Error(fmt.Sprintf("写入文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			if isPermissionError(err) {
				return tools.Error(fmt.Sprintf("无权限写入 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
			}
			return tools.Error(fmt.Sprintf("写入文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
		}
	}

	tmpPath := fullPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(contentText), 0o644); err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限写入 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("写入文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限写入 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("写入文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	data := map[string]any{
		"written":     true,
		"size_bytes":  len([]byte(contentText)),
		"backup_path": relOrOriginal(backupPath, t.WorkingDir),
	}
	return tools.Success(fmt.Sprintf("成功写入 %s (%d 字节)", path, len([]byte(contentText))), data)
}

func (t *WriteTool) backupFile(fullPath string) (string, error) {
	backupDir := filepath.Join(filepath.Dir(fullPath), ".backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(fullPath), timestamp)
	backupPath := filepath.Join(backupDir, backupName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return "", err
	}
	return backupPath, nil
}

func (t *WriteTool) resolvePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(normalized) {
		return normalized
	}
	return filepath.Join(t.WorkingDir, normalized)
}

type EditTool struct {
	tools.BaseTool
	ProjectRoot string
	WorkingDir  string
	Registry    *tools.ToolRegistry
}

func NewEditTool(projectRoot string) *EditTool {
	return NewEditToolWithOptions(projectRoot, "", nil)
}

func NewEditToolWithOptions(projectRoot string, workingDir string, registry *tools.ToolRegistry) *EditTool {
	if projectRoot == "" {
		projectRoot = "."
	}
	absRoot, _ := filepath.Abs(projectRoot)
	absWorkingDir := absRoot
	if workingDir != "" {
		absWorkingDir, _ = filepath.Abs(workingDir)
	}
	base := tools.NewBaseTool("Edit", "精确替换文件内容，支持冲突检测和自动备份", false)
	base.Parameters = map[string]tools.ToolParameter{
		"path":          {Name: "path", Type: "string", Description: "要编辑的文件路径（相对项目根目录）", Required: true},
		"old_string":    {Name: "old_string", Type: "string", Description: "要替换的内容（必须唯一匹配）", Required: true},
		"new_string":    {Name: "new_string", Type: "string", Description: "替换后的内容", Required: true},
		"file_mtime_ms": {Name: "file_mtime_ms", Type: "integer", Description: "缓存的文件修改时间（用于冲突检测）", Required: false},
	}
	t := &EditTool{
		BaseTool:    base,
		ProjectRoot: absRoot,
		WorkingDir:  absWorkingDir,
		Registry:    registry,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *EditTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *EditTool) Run(parameters map[string]any) tools.ToolResponse {
	path, _ := parameters["path"].(string)
	oldString, oldOk := parameters["old_string"]
	newString, newOk := parameters["new_string"]
	rawCachedMtime, hasCachedMtime := parameters["file_mtime_ms"]
	cachedMtime := int64(intFromAny(rawCachedMtime))

	if path == "" {
		return tools.Error("缺少必需参数: path", tools.ToolErrorCodeInvalidParam, nil)
	}
	if !oldOk {
		return tools.Error("缺少必需参数: old_string", tools.ToolErrorCodeInvalidParam, nil)
	}
	if !newOk {
		return tools.Error("缺少必需参数: new_string", tools.ToolErrorCodeInvalidParam, nil)
	}

	oldText := fmt.Sprintf("%v", oldString)
	newText := fmt.Sprintf("%v", newString)
	fullPath := t.resolvePath(path)

	st, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tools.Error(fmt.Sprintf("文件 '%s' 不存在", path), tools.ToolErrorCodeNotFound, nil)
		}
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("编辑文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	currentMtime := st.ModTime().UnixMilli()
	if hasCachedMtime && currentMtime != cachedMtime {
		return tools.Error(
			fmt.Sprintf("文件自上次读取后被修改。当前 mtime=%d, 缓存 mtime=%d", currentMtime, cachedMtime),
			tools.ToolErrorCodeConflict,
			map[string]any{"current_mtime_ms": currentMtime, "cached_mtime_ms": cachedMtime},
		)
	}

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("编辑文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}
	content := string(contentBytes)
	matches := strings.Count(content, oldText)
	if matches != 1 {
		return tools.Error(
			fmt.Sprintf("old_string 必须唯一匹配文件内容。找到 %d 处匹配。", matches),
			tools.ToolErrorCodeInvalidParam,
			map[string]any{"matches": matches},
		)
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	backupPath, err := t.backupFile(fullPath)
	if err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("编辑文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}
	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("编辑文件失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	changedBytes := len([]byte(newText)) - len([]byte(oldText))
	return tools.Success(
		fmt.Sprintf("成功编辑 %s (变化 %+d 字节)", path, changedBytes),
		map[string]any{
			"modified":      true,
			"changed_bytes": changedBytes,
			"backup_path":   relOrOriginal(backupPath, t.WorkingDir),
		},
	)
}

func (t *EditTool) backupFile(fullPath string) (string, error) {
	backupDir := filepath.Join(filepath.Dir(fullPath), ".backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(fullPath), timestamp)
	backupPath := filepath.Join(backupDir, backupName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return "", err
	}
	return backupPath, nil
}

func (t *EditTool) resolvePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(normalized) {
		return normalized
	}
	return filepath.Join(t.WorkingDir, normalized)
}

type MultiEditTool struct {
	tools.BaseTool
	ProjectRoot string
	WorkingDir  string
	Registry    *tools.ToolRegistry
}

func NewMultiEditTool(projectRoot string) *MultiEditTool {
	return NewMultiEditToolWithOptions(projectRoot, "", nil)
}

func NewMultiEditToolWithOptions(projectRoot string, workingDir string, registry *tools.ToolRegistry) *MultiEditTool {
	if projectRoot == "" {
		projectRoot = "."
	}
	absRoot, _ := filepath.Abs(projectRoot)
	absWorkingDir := absRoot
	if workingDir != "" {
		absWorkingDir, _ = filepath.Abs(workingDir)
	}
	base := tools.NewBaseTool("MultiEdit", "批量替换文件内容，支持原子性和冲突检测", false)
	base.Parameters = map[string]tools.ToolParameter{
		"path":          {Name: "path", Type: "string", Description: "要编辑的文件路径（相对项目根目录）", Required: true},
		"edits":         {Name: "edits", Type: "array", Description: "替换列表，每项包含 old_string 和 new_string", Required: true},
		"file_mtime_ms": {Name: "file_mtime_ms", Type: "integer", Description: "缓存的文件修改时间（用于冲突检测）", Required: false},
	}
	t := &MultiEditTool{
		BaseTool:    base,
		ProjectRoot: absRoot,
		WorkingDir:  absWorkingDir,
		Registry:    registry,
	}
	t.BaseTool.SetRunImpl(t.Run)
	return t
}

func (t *MultiEditTool) GetParameters() []tools.ToolParameter {
	return t.BaseTool.GetParameters()
}

func (t *MultiEditTool) Run(parameters map[string]any) tools.ToolResponse {
	path, _ := parameters["path"].(string)
	rawEdits, ok := parameters["edits"].([]any)
	rawCachedMtime, hasCachedMtime := parameters["file_mtime_ms"]
	cachedMtime := int64(intFromAny(rawCachedMtime))

	if path == "" {
		return tools.Error("缺少必需参数: path", tools.ToolErrorCodeInvalidParam, nil)
	}
	if !ok || len(rawEdits) == 0 {
		return tools.Error("缺少必需参数: edits（必须是列表）", tools.ToolErrorCodeInvalidParam, nil)
	}

	fullPath := t.resolvePath(path)
	st, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tools.Error(fmt.Sprintf("文件 '%s' 不存在", path), tools.ToolErrorCodeNotFound, nil)
		}
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("批量编辑失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	currentMtime := st.ModTime().UnixMilli()
	if hasCachedMtime && currentMtime != cachedMtime {
		return tools.Error(
			fmt.Sprintf("文件自上次读取后被修改。所有替换已取消。当前 mtime=%d, 缓存 mtime=%d", currentMtime, cachedMtime),
			tools.ToolErrorCodeConflict,
			map[string]any{"current_mtime_ms": currentMtime, "cached_mtime_ms": cachedMtime},
		)
	}

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("批量编辑失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}
	content := string(contentBytes)
	original := content

	type editOp struct {
		Old string
		New string
	}
	edits := make([]editOp, 0, len(rawEdits))

	for i, raw := range rawEdits {
		em, _ := raw.(map[string]any)
		if em == nil {
			return tools.Error(fmt.Sprintf("编辑项 %d 缺少 old_string 或 new_string", i), tools.ToolErrorCodeInvalidParam, nil)
		}
		oldRaw, oldExists := em["old_string"]
		newRaw, newExists := em["new_string"]
		if !oldExists || !newExists || oldRaw == nil || newRaw == nil {
			return tools.Error(fmt.Sprintf("编辑项 %d 缺少 old_string 或 new_string", i), tools.ToolErrorCodeInvalidParam, nil)
		}
		oldStr := fmt.Sprintf("%v", oldRaw)
		newStr := fmt.Sprintf("%v", newRaw)
		matches := strings.Count(content, oldStr)
		if matches != 1 {
			return tools.Error(
				fmt.Sprintf("编辑项 %d: old_string 必须唯一匹配。找到 %d 处匹配。", i, matches),
				tools.ToolErrorCodeInvalidParam,
				map[string]any{"edit_index": i, "matches": matches},
			)
		}
		edits = append(edits, editOp{Old: oldStr, New: newStr})
	}

	for _, e := range edits {
		content = strings.Replace(content, e.Old, e.New, 1)
	}

	backupPath, err := t.backupFile(fullPath)
	if err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("批量编辑失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		if isPermissionError(err) {
			return tools.Error(fmt.Sprintf("无权限编辑 '%s'", path), tools.ToolErrorCodePermissionDenied, nil)
		}
		return tools.Error(fmt.Sprintf("批量编辑失败：%v", err), tools.ToolErrorCodeInternalError, nil)
	}

	changedBytes := len([]byte(content)) - len([]byte(original))
	return tools.Success(
		fmt.Sprintf("成功执行 %d 个替换操作 (变化 %+d 字节)", len(edits), changedBytes),
		map[string]any{
			"modified":      true,
			"num_edits":     len(edits),
			"changed_bytes": changedBytes,
			"backup_path":   relOrOriginal(backupPath, t.WorkingDir),
		},
	)
}

func (t *MultiEditTool) backupFile(fullPath string) (string, error) {
	backupDir := filepath.Join(filepath.Dir(fullPath), ".backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(fullPath), timestamp)
	backupPath := filepath.Join(backupDir, backupName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return "", err
	}
	return backupPath, nil
}

func (t *MultiEditTool) resolvePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(normalized) {
		return normalized
	}
	return filepath.Join(t.WorkingDir, normalized)
}

func formatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	value := float64(size)
	for _, unit := range units {
		if value < 1024 {
			return fmt.Sprintf("%.1f%s", value, unit)
		}
		value /= 1024
	}
	return fmt.Sprintf("%.1fTB", value)
}

func formatTime(ts time.Time) string {
	return ts.Format("2006-01-02 15:04:05")
}

func relOrOriginal(path string, base string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case bool:
		if n {
			return 1
		}
		return 0
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func isPermissionError(err error) bool {
	return errors.Is(err, fs.ErrPermission)
}
