# 开发日志系统指南（DevLog System）

## 📖 概述

**DevLogTool** 是 HelloAgents-Go 框架的结构化开发日志工具，用于记录 Agent 的开发决策、问题、解决方案等关键信息。

### 核心特性

- ✅ **结构化日志**：category + content + metadata
- ✅ **7 种类别**：decision、progress、issue、solution、refactor、test、performance
- ✅ **持久化存储**：保存到 `memory/devlogs/`
- ✅ **过滤查询**：按类别、标签查询
- ✅ **自动摘要**：生成日志摘要

---

## 🚀 快速开始

### 1. 自动集成（零配置）

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

// DevLogTool 默认启用
config := core.DefaultConfig()
config.DevlogEnabled = true

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", nil, config, nil, 0, nil)

// Agent 可以直接使用 DevLog 工具
agent.Run("记录开发决策：使用 Redis 作为缓存", nil)
```

### 2. 手动使用

```go
import "helloagents-go/hello_agents/tools/builtin"

tool := builtin.NewDevLogTool("session-001", "assistant", "./", "memory/devlogs")

// 记录决策
tool.Run(map[string]any{
    "category": "decision",
    "content":  "选择 Redis 作为缓存方案",
    "metadata": map[string]any{
        "reason":       "高性能、支持持久化",
        "alternatives": []string{"Memcached", "本地缓存"},
    },
})

// 记录问题
tool.Run(map[string]any{
    "category": "issue",
    "content":  "数据库连接池耗尽",
    "metadata": map[string]any{
        "severity": "high",
        "impact":   "API 响应超时",
    },
})

// 记录解决方案
tool.Run(map[string]any{
    "category": "solution",
    "content":  "增加连接池大小到 50",
    "metadata": map[string]any{
        "issue_id": "db-pool-exhausted",
        "result":   "问题解决",
    },
})
```

---

## 💡 核心概念

### 7 种日志类别

| 类别          | 用途     | 示例                 |
| ------------- | -------- | -------------------- |
| `decision`    | 技术决策 | 选择数据库、架构设计 |
| `progress`    | 进度更新 | 完成模块、里程碑     |
| `issue`       | 问题记录 | Bug、性能问题、错误  |
| `solution`    | 解决方案 | 问题修复、优化方案   |
| `refactor`    | 重构记录 | 代码重构、架构调整   |
| `test`        | 测试记录 | 测试结果、覆盖率     |
| `performance` | 性能分析 | 性能瓶颈、优化效果   |

### 日志结构

```json
{
  "id": "devlog-20250220-103045",
  "timestamp": "2026-02-21T10:30:45Z",
  "category": "decision",
  "content": "选择 Redis 作为缓存方案",
  "metadata": {
    "reason": "高性能、支持持久化",
    "alternatives": ["Memcached", "本地缓存"],
    "tags": ["cache", "redis"]
  }
}
```

---

## 📝 使用指南

### 1. 记录不同类型的日志

**决策日志：**
```go
tool.Run(map[string]any{
    "category": "decision",
    "content":  "使用 PostgreSQL 作为主数据库",
    "metadata": map[string]any{
        "reason":       "支持 JSONB、事务完整性",
        "alternatives": []string{"MySQL", "MongoDB"},
        "tags":         []string{"database", "architecture"},
    },
})
```

**进度日志：**
```go
tool.Run(map[string]any{
    "category": "progress",
    "content":  "完成用户认证模块",
    "metadata": map[string]any{
        "milestone":  "v1.0",
        "completion": "80%",
    },
})
```

**性能日志：**
```go
tool.Run(map[string]any{
    "category": "performance",
    "content":  "API 响应时间优化",
    "metadata": map[string]any{
        "before":      "500ms",
        "after":       "150ms",
        "improvement": "70%",
    },
})
```

### 2. 查询日志

```go
// 查询所有日志
tool.Run(map[string]any{"action": "list"})

// 按类别查询
tool.Run(map[string]any{"action": "list", "category": "issue"})

// 生成摘要
tool.Run(map[string]any{"action": "summary"})
```

### 3. 清空日志

```go
tool.Run(map[string]any{"action": "clear"})
```

---

## 📊 实际案例

### 案例 1：问题追踪

```go
// 1. 记录问题
tool.Run(map[string]any{
    "category": "issue",
    "content":  "数据库查询慢，响应时间 > 2s",
    "metadata": map[string]any{"severity": "high"},
})

// 2. 记录分析
tool.Run(map[string]any{
    "category": "performance",
    "content":  "缺少索引导致全表扫描",
})

// 3. 记录解决方案
tool.Run(map[string]any{
    "category": "solution",
    "content":  "添加 email 字段索引",
    "metadata": map[string]any{
        "before":      "2.3s",
        "after":       "0.05s",
        "improvement": "97.8%",
    },
})
```

---

## 🎯 最佳实践

### 1. 使用标签组织日志

```go
tool.Run(map[string]any{
    "category": "issue",
    "content":  "内存泄漏",
    "metadata": map[string]any{
        "tags": []string{"memory", "bug", "critical"},
    },
})
```

### 2. 关联相关日志

```go
// 记录问题时使用 issue_id
tool.Run(map[string]any{
    "category": "issue",
    "content":  "数据库连接池耗尽",
    "metadata": map[string]any{"issue_id": "db-pool-001"},
})

// 解决方案引用问题 ID
tool.Run(map[string]any{
    "category": "solution",
    "content":  "增加连接池大小",
    "metadata": map[string]any{"issue_id": "db-pool-001", "result": "问题解决"},
})
```

---

## 🔗 相关文档

- [日志系统](./logging-system-guide.md) - 四种日志范式对比
- [可观测性](./observability-guide.md) - TraceLogger 使用
- [TodoWrite](./todowrite-usage-guide.md) - 任务进度管理

---

## ❓ 常见问题

**Q: DevLogTool 和 TraceLogger 的区别？**

A:
- **DevLogTool**: 记录开发决策、问题、解决方案（结构化）
- **TraceLogger**: 记录执行轨迹、工具调用、LLM 请求（审计）

**Q: 如何禁用 DevLogTool？**

A: 设置 `DevlogEnabled = false`：
```go
config.DevlogEnabled = false
```

**Q: 日志文件在哪里？**

A: 默认保存在 `memory/devlogs/` 目录。

---

**最后更新**: 2026-02-21
