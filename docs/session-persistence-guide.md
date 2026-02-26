# 会话持久化使用指南

> 断点续跑的秘密——保存会话状态，随时恢复。

---

## 📖 概述

HelloAgents-Go 的会话持久化功能允许你：

- ✅ **保存会话**：将 Agent 的完整状态保存到文件
- ✅ **恢复会话**：从文件恢复，实现断点续跑
- ✅ **环境检查**：自动检测配置和工具变化
- ✅ **异常保护**：崩溃或中断时自动保存
- ✅ **团队协作**：共享会话文件，多人协作

---

## 🚀 快速开始

### 基本使用

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

// 创建 Agent（默认启用会话持久化）
config := core.DefaultConfig()
config.SessionEnabled = true

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewSimpleAgent("assistant", llm, "你是一个有用的AI助手", nil, config)

// 正常使用
result, _ := agent.Run("帮我分析这个项目", nil)

// 手动保存会话
filepath := agent.SaveSession("my-analysis-session")
fmt.Printf("会话已保存: %s\n", filepath)

// 恢复会话
agent.LoadSession("memory/sessions/my-analysis-session.json", true)

// 列出所有会话
sessions := agent.ListSessions()
for _, s := range sessions {
    fmt.Printf("%s - %s\n", s["session_id"], s["saved_at"])
}
```

---

## 📋 核心功能

### 1. 保存会话

```go
filepath := agent.SaveSession("my-session-name")
// 保存到: memory/sessions/my-session-name.json
```

**会话快照包含**：
- 完整的对话历史
- Agent 配置信息
- 工具 Schema 哈希
- Read 工具的文件元数据缓存
- 统计信息（tokens、steps、duration）

### 2. 恢复会话

```go
// 加载会话（默认检查环境一致性）
agent.LoadSession("memory/sessions/my-session-name.json", true)

// 跳过一致性检查
agent.LoadSession("memory/sessions/my-session-name.json", false)
```

### 3. 列出会话

```go
sessions := agent.ListSessions()
for _, session := range sessions {
    fmt.Printf("会话 ID: %s\n", session["session_id"])
    fmt.Printf("保存时间: %s\n", session["saved_at"])
}
```

---

## ⚙️ 配置选项

```go
config := core.DefaultConfig()
config.SessionEnabled = true                // 是否启用
config.SessionDir = "memory/sessions"       // 保存目录
config.AutoSaveEnabled = false              // 是否自动保存
config.AutoSaveInterval = 10                // 自动保存间隔（每 N 条消息）
```

---

## 🛡️ 异常保护（ReActAgent）

ReActAgent 自动在异常时保存会话：

```go
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 使用 defer + recover 处理 panic
defer func() {
    if r := recover(); r != nil {
        // 自动保存为 session-error.json
        agent.SaveSession("session-error")
    }
}()

result, err := agent.Run("长时间任务", nil)
if err != nil {
    // 自动保存为 session-error.json
    agent.SaveSession("session-error")
}
```

---

## 📦 会话文件结构

```json
{
  "session_id": "s-20250119-a3f2d8e1",
  "created_at": "2025-01-19T10:30:45Z",
  "saved_at": "2025-01-19T10:45:10Z",
  "agent_config": {
    "name": "assistant",
    "agent_type": "ReActAgent",
    "llm_provider": "openai",
    "llm_model": "gpt-4",
    "max_steps": 10
  },
  "history": [...],
  "tool_schema_hash": "a3f2d8e1",
  "read_cache": {...},
  "metadata": {
    "total_tokens": 43500,
    "total_steps": 25,
    "duration_seconds": 877
  }
}
```

---

## 💡 使用场景

### 场景 1：长时间任务断点续跑

```go
// 第一次运行
agent.Run("分析整个代码库的架构", nil)
// 网络断开，自动保存为 session-error.json

// 恢复后继续
agent.LoadSession("memory/sessions/session-error.json", true)
agent.Run("继续之前的分析", nil)
```

### 场景 2：团队协作

```go
// 第一个人
agent1.Run("开始分析项目", nil)
agent1.SaveSession("team-analysis-session")

// 第二个人接手
agent2.LoadSession("memory/sessions/team-analysis-session.json", true)
agent2.Run("继续分析", nil)
```

---

## 🔧 高级用法

### 编程式访问 SessionStore

```go
import "helloagents-go/hello_agents/core"

store := core.NewSessionStore("my-sessions")

// 列出所有会话
sessions := store.ListSessions()

// 加载会话数据
sessionData := store.Load("my-sessions/my-session.json")

// 检查一致性
configCheck := store.CheckConfigConsistency(savedConfig, currentConfig)
toolCheck := store.CheckToolSchemaConsistency(savedHash, currentHash)
```

---

## 🎯 最佳实践

1. **定期保存**：长时间任务建议启用自动保存
2. **命名规范**：使用有意义的会话名称（如 `project-analysis-2025-01-19`）
3. **清理旧会话**：定期删除不需要的会话文件
4. **版本控制**：不要提交会话文件到 Git
5. **团队协作**：通过其他方式共享会话文件（如云存储）

---

## 🔗 相关文档

- [上下文工程](./context-engineering-guide.md) - 历史压缩和管理
- [文件工具](./file_tools.md) - 乐观锁机制
- [可观测性](./observability-guide.md) - 追踪会话执行

---

**最后更新**：2026-02-21
**维护者**：HelloAgents 开发团队
