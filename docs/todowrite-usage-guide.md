# TodoWrite 进度管理工具使用指南

> 提供任务列表管理能力，强制单线程专注，避免任务切换

---

## 📚 目录

- [快速开始](#快速开始)
- [核心特性](#核心特性)
- [使用示例](#使用示例)
- [API 参考](#api-参考)
- [最佳实践](#最佳实践)
- [实战案例](#实战案例)

---

## 快速开始

### 零配置使用（推荐）

TodoWrite 工具已内置在 HelloAgents-Go 框架中，默认启用。

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
)

// 创建 Agent（TodoWriteTool 会自动注册）
config := core.DefaultConfig()
config.TodowriteEnabled = true

registry := tools.NewToolRegistry(nil)
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)

agent, _ := agents.NewReActAgent("开发助手", llm, "", registry, config, nil, 0, nil)

// Agent 可以直接使用 TodoWrite 工具
agent.Run("帮我实现用户系统、订单系统和支付系统", nil)
```

### 手动使用

```go
import "helloagents-go/hello_agents/tools/builtin"

// 创建工具
tool := builtin.NewTodoWriteTool("./", "memory/todos")

// 创建任务列表
response := tool.Run(map[string]any{
    "summary": "实现电商核心功能",
    "todos": []map[string]any{
        {"content": "实现用户认证", "status": "pending"},
        {"content": "实现订单处理", "status": "pending"},
        {"content": "实现支付功能", "status": "pending"},
    },
})

fmt.Println(response.Text)
// 📋 [0/3] 待处理: 实现用户认证; 实现订单处理; 实现支付功能
```

---

## 核心特性

### 1. 声明式覆盖

每次提交完整的任务列表，避免状态不一致。

```go
// ✅ 正确：提交完整列表
response := tool.Run(map[string]any{
    "todos": []map[string]any{
        {"content": "任务1", "status": "completed"},
        {"content": "任务2", "status": "in_progress"},
        {"content": "任务3", "status": "pending"},
    },
})
```

### 2. 单线程强制

最多只能有 1 个任务标记为 `in_progress`，防止任务切换和焦点丢失。

```go
// ❌ 错误：多个 in_progress
response := tool.Run(map[string]any{
    "todos": []map[string]any{
        {"content": "任务1", "status": "in_progress"},
        {"content": "任务2", "status": "in_progress"}, // 违反约束
    },
})
// 返回错误：最多只能有 1 个 in_progress 任务
```

### 3. 自动 Recap 生成

```
// 部分完成
"📋 [2/5] 进行中: 实现订单查询. 待处理: 实现订单创建; 实现订单更新"

// 全部完成
"✅ [5/5] 所有任务已完成！"
```

### 4. 持久化支持

任务列表自动保存到文件，支持断点恢复。

---

## 使用示例

### 示例 1：基本工作流

```go
tool := builtin.NewTodoWriteTool("./", "memory/todos")

// 1. 创建任务列表
tool.Run(map[string]any{
    "summary": "实现博客系统",
    "todos": []map[string]any{
        {"content": "设计数据库", "status": "pending"},
        {"content": "实现用户模块", "status": "pending"},
        {"content": "实现文章模块", "status": "pending"},
    },
})
// 📋 [0/3] 待处理: 设计数据库; 实现用户模块; 实现文章模块

// 2. 开始第一个任务
tool.Run(map[string]any{
    "todos": []map[string]any{
        {"content": "设计数据库", "status": "in_progress"},
        {"content": "实现用户模块", "status": "pending"},
        {"content": "实现文章模块", "status": "pending"},
    },
})

// 3. 完成第一个，开始第二个
tool.Run(map[string]any{
    "todos": []map[string]any{
        {"content": "设计数据库", "status": "completed"},
        {"content": "实现用户模块", "status": "in_progress"},
        {"content": "实现文章模块", "status": "pending"},
    },
})

// 4. 全部完成
tool.Run(map[string]any{
    "todos": []map[string]any{
        {"content": "设计数据库", "status": "completed"},
        {"content": "实现用户模块", "status": "completed"},
        {"content": "实现文章模块", "status": "completed"},
    },
})
// ✅ [3/3] 所有任务已完成！
```

### 示例 2：清空任务列表

```go
response := tool.Run(map[string]any{"action": "clear"})
// ✅ 任务列表已清空
```

---

## API 参考

### TodoWriteTool

```go
func NewTodoWriteTool(projectRoot string, persistenceDir string) *TodoWriteTool
```

**参数**：
- `projectRoot`: 项目根目录
- `persistenceDir`: 持久化目录（相对于 projectRoot）

### Run() 方法

```go
func (t *TodoWriteTool) Run(params map[string]any) *tools.ToolResponse
```

**参数**：
- `summary` (string, 可选): 总体任务描述
- `todos` ([]map[string]any, 可选): 待办事项列表
- `action` (string, 可选): 操作类型（create/update/clear）

**todos 格式**：
```go
[]map[string]any{
    {"content": "任务内容", "status": "pending"},   // pending | in_progress | completed
}
```

---

## 最佳实践

### 1. 任务粒度

✅ **推荐**：适中粒度，每个任务 1-4 小时

### 2. 任务数量

- **建议**：5-10 个任务为宜
- **最多**：不超过 20 个

### 3. 状态转换

遵循单向流转：`pending → in_progress → completed`

---

## 配置选项

```go
config := core.DefaultConfig()
config.TodowriteEnabled = true                     // 启用/禁用
config.TodowritePersistenceDir = "memory/todos"    // 持久化目录
```

---

## 常见问题

### Q1: 可以同时进行多个任务吗？

A: 不可以。TodoWrite 强制单线程，最多 1 个 `in_progress`。

### Q2: 任务列表会自动保存吗？

A: 是的，每次调用 `Run()` 都会自动保存。

---

## 相关文档

- [工具响应协议](./tool-response-protocol.md)
- [会话持久化](./session-persistence-guide.md)
- [子代理机制](./subagent-guide.md)

---

## 示例代码

完整示例代码请参考：
- `example/todowrite_demo/main.go` - 基础示例
- `example/todowrite_real_world/main.go` - 实战案例
