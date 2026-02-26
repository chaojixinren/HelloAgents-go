# HelloAgents-Go 文件操作工具使用指南

> 提供标准的文件读写编辑能力，内置乐观锁机制，确保多进程/多 Agent 协作时的数据安全

---

## 📚 目录

- [快速开始](#快速开始)
- [工具介绍](#工具介绍)
- [乐观锁机制](#乐观锁机制)
- [使用示例](#使用示例)
- [API 参考](#api-参考)
- [最佳实践](#最佳实践)

---

## 快速开始

### 安装

文件工具已内置在 HelloAgents-Go 框架中，无需额外安装。

### 基本使用

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
    "helloagents-go/hello_agents/tools/builtin"
)

// 1. 创建工具注册表
registry := tools.NewToolRegistry(nil)

// 2. 注册文件工具
registry.RegisterTool(builtin.NewReadTool("./", registry), false)
registry.RegisterTool(builtin.NewWriteTool("./"), false)
registry.RegisterTool(builtin.NewEditTool("./"), false)

// 3. 创建 Agent
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 4. Agent 自动使用文件工具
result, _ := agent.Run("读取 config.go，然后修改 API_KEY 为 'new_key_123'", nil)
```

---

## 工具介绍

HelloAgents-Go 提供 3 个专业的文件操作工具：

### 1. ReadTool - 文件读取

**功能**：
- 读取文件内容
- 支持行号范围（offset/limit）
- 自动获取文件元数据（mtime, size）
- 缓存元数据到 ToolRegistry（用于乐观锁）

**参数**：
- `path` (必需): 文件路径（相对于 project_root）
- `offset` (可选): 起始行号，默认 0
- `limit` (可选): 最大行数，默认 2000

### 2. WriteTool - 文件写入

**功能**：
- 创建或覆盖文件
- 乐观锁冲突检测（如果文件已存在）
- 原子写入（临时文件 + rename）
- 自动备份原文件

**参数**：
- `path` (必需): 文件路径
- `content` (必需): 文件内容
- `file_mtime_ms` (可选): 缓存的 mtime（用于冲突检测）

### 3. EditTool - 精确替换

**功能**：
- 精确替换文件内容（old_string 必须唯一匹配）
- 乐观锁冲突检测
- 自动备份原文件

**参数**：
- `path` (必需): 文件路径
- `old_string` (必需): 要替换的内容
- `new_string` (必需): 替换后的内容
- `file_mtime_ms` (可选): 缓存的 mtime

---

## 乐观锁机制

### 什么是乐观锁？

乐观锁是一种并发控制机制，通过检测文件是否在读取后被修改，来避免意外覆盖。

### 工作原理

```
1. Read("config.go")
   ├─ 读取文件内容
   ├─ 获取元数据（mtime=123456, size=4217）
   └─ 缓存到 ToolRegistry

2. [外部修改 config.go]
   └─ mtime 变为 123789

3. Edit("config.go", file_mtime_ms=123456)
   ├─ 检查当前 mtime (123789) vs 缓存 mtime (123456)
   ├─ 不一致 → 返回 CONFLICT 错误
   └─ Agent 看到冲突，重新 Read
```

---

## 使用示例

### 示例 1：基本文件操作

```go
registry := tools.NewToolRegistry(nil)
readTool := builtin.NewReadTool("./", registry)
writeTool := builtin.NewWriteTool("./")
editTool := builtin.NewEditTool("./")

// 1. 写入文件
response := writeTool.Run(map[string]any{
    "path":    "config.go",
    "content": "package config\n\nconst APIKey = \"test_key\"\nconst Debug = false\n",
})
fmt.Println(response.Text) // 成功写入 config.go (XX 字节)

// 2. 读取文件
response = readTool.Run(map[string]any{"path": "config.go"})
fmt.Println(response.Data["content"])

// 3. 编辑文件
response = editTool.Run(map[string]any{
    "path":       "config.go",
    "old_string": "const Debug = false",
    "new_string": "const Debug = true",
})
fmt.Println(response.Text) // 成功编辑 config.go
```

### 示例 2：乐观锁冲突检测

```go
// 1. Agent 读取文件（缓存元数据）
response := readTool.Run(map[string]any{"path": "data.txt"})
fmt.Printf("缓存的 mtime: %v\n", response.Data["file_mtime_ms"])

// 2. 模拟外部修改
os.WriteFile("data.txt", []byte("Modified by external process"), 0644)

// 3. Agent 尝试编辑（使用缓存的 mtime）
cachedMeta := registry.GetReadMetadata("data.txt")
response = editTool.Run(map[string]any{
    "path":          "data.txt",
    "old_string":    "Original content",
    "new_string":    "My changes",
    "file_mtime_ms": cachedMeta["file_mtime_ms"],
})

// 检测到冲突！
if response.Status == tools.StatusError {
    fmt.Printf("✅ 冲突检测成功: %s\n", response.ErrorInfo["message"])
}
```

### 示例 3：在 Agent 中使用

```go
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(builtin.NewReadTool("./", registry), false)
registry.RegisterTool(builtin.NewWriteTool("./"), false)
registry.RegisterTool(builtin.NewEditTool("./"), false)

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// Agent 自动使用乐观锁
result, _ := agent.Run(`
请执行以下任务：
1. 读取 config.go 文件
2. 将 APIKey 修改为 'new_key_456'
3. 将 Debug 修改为 true
`, nil)
```

---

## API 参考

### ReadTool

```go
func NewReadTool(projectRoot string, registry *tools.ToolRegistry) *ReadTool
```

**参数**：
- `projectRoot`: 项目根目录
- `registry`: ToolRegistry 实例（用于元数据缓存）

### WriteTool

```go
func NewWriteTool(projectRoot string) *WriteTool
```

### EditTool

```go
func NewEditTool(projectRoot string) *EditTool
```

---

## 最佳实践

### 1. 始终传递 registry

```go
// ✅ 推荐：传递 registry，启用乐观锁
readTool := builtin.NewReadTool("./", registry)

// ❌ 不推荐：不传递 registry，无法使用乐观锁
readTool := builtin.NewReadTool("./", nil)
```

### 2. Read 后再 Edit

```go
// ✅ 推荐：先 Read，缓存元数据
readTool.Run(map[string]any{"path": "config.go"})
editTool.Run(map[string]any{
    "path":       "config.go",
    "old_string": "old",
    "new_string": "new",
})
```

### 3. 处理冲突错误

```go
response := editTool.Run(params)
if response.Status == tools.StatusError {
    if response.ErrorInfo["code"] == "CONFLICT" {
        // 冲突：重新读取文件
        readTool.Run(map[string]any{"path": "config.go"})
        // 然后重试编辑
    }
}
```

---

## 常见问题

### Q1: 为什么 Edit 返回 "old_string 必须唯一匹配" 错误？

**原因**：EditTool 要求 `old_string` 在文件中只出现一次，以确保替换的精确性。

**解决**：使用更具体的 `old_string`，包含更多上下文。

### Q2: 如何禁用乐观锁？

**方法**：不传递 `file_mtime_ms` 参数即可。

### Q3: 跨平台兼容性如何？

**答**：完全兼容 Windows、Linux、macOS，使用 `filepath` 包统一路径处理。

---

## 相关文档

- [工具响应协议](./tool-response-protocol.md)
- [熔断器机制](./circuit-breaker-guide.md)
- [可观测性](./observability-guide.md)

---

**最后更新**：2026-02-21
**维护者**：HelloAgents 开发团队
