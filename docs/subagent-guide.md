# 子代理机制指南（Subagent Mechanism）

## 📖 概述

**子代理机制**允许主 Agent 将复杂任务分解为子任务，委派给独立的子 Agent 执行，实现上下文隔离和工具权限控制。

### 核心特性

- ✅ **上下文隔离**：子代理使用独立历史，不污染主 Agent
- ✅ **工具过滤**：限制子代理可用工具（只读、完全访问、自定义）
- ✅ **灵活组合**：所有 Agent 类型都可作为子代理
- ✅ **成本优化**：子任务可用轻量模型（节省 70%）
- ✅ **零配置**：TaskTool 自动注册

---

## 🚀 快速开始

### 1. 零配置使用（推荐）

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

// 启用子代理机制
config := core.DefaultConfig()
config.SubagentEnabled = true

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("main", llm, "", nil, config, nil, 0, nil)

// TaskTool 已自动注册，Agent 可以直接使用
agent.Run("使用 Task 工具探索项目结构", nil)
```

### 2. 手动调用子代理

```go
import "helloagents-go/hello_agents/tools"

// 创建主 Agent 和子 Agent
mainAgent, _ := agents.NewReActAgent("main", llm, "", registry, nil, nil, 0, nil)
exploreAgent, _ := agents.NewReActAgent("explorer", llm, "", registry, nil, nil, 0, nil)

// 手动调用子代理（上下文隔离）
result := exploreAgent.RunAsSubagent(
    "探索 hello_agents/core/ 目录",
    tools.NewReadOnlyFilter(),  // 只读权限
    true,                        // 返回摘要
)

fmt.Printf("子代理结果: %s\n", result["summary"])
```

---

## 💡 核心概念

### 1. 上下文隔离

```go
// ✅ 好：上下文隔离
mainAgent.Run("分析项目", nil)  // 主任务

// 子任务 1：探索（独立历史）
exploreAgent.RunAsSubagent("探索项目结构", readOnlyFilter, true)

// 子任务 2：分析（独立历史）
analyzeAgent.RunAsSubagent("分析架构设计", readOnlyFilter, true)

// 主 Agent 历史保持清晰
```

### 2. 工具过滤

**3 种内置过滤器：**

```go
import "helloagents-go/hello_agents/tools"

readOnly := tools.NewReadOnlyFilter()      // 只读工具
fullAccess := tools.NewFullAccessFilter()   // 完全访问（排除危险工具）
custom := tools.NewCustomFilter(            // 自定义白名单/黑名单
    []string{"Read", "Search"},
    nil,
    "whitelist",
)
```

**ReadOnlyFilter（只读）：**
```go
readonly := tools.NewReadOnlyFilter()
allowed := readonly.Filter([]string{"Read", "Write", "Bash", "Search"})
// 返回：["Read", "Search"]
```

**FullAccessFilter（完全访问）：**
```go
full := tools.NewFullAccessFilter()
allowed := full.Filter([]string{"Read", "Write", "Bash", "Terminal"})
// 返回：["Read", "Write"]  // 排除 Bash, Terminal
```

### 3. Agent 工厂

```go
import "helloagents-go/hello_agents/agents"

// 创建不同类型的 Agent
reactAgent := agents.CreateAgent("react", "explorer", llm, registry)
reflectionAgent := agents.CreateAgent("reflection", "thinker", llm, registry)
planAgent := agents.CreateAgent("plan", "planner", llm, registry)
simpleAgent := agents.CreateAgent("simple", "assistant", llm, registry)
```

---

## 📝 使用指南

### 1. TaskTool 参数

```go
// Agent 调用 TaskTool 的参数
params := map[string]any{
    "task":        "任务描述",
    "agent_type":  "react",    // react / reflection / plan / simple
    "tool_filter": "readonly", // readonly / full / none
    "max_steps":   15,         // 最大步数（可选）
}
```

### 2. 不同类型的子代理

```go
// 探索任务 → ReActAgent（快速迭代）
agents.CreateAgent("react", "explorer", llm, registry)

// 深度分析 → ReflectionAgent（反思优化）
agents.CreateAgent("reflection", "analyzer", llm, registry)

// 规划任务 → PlanAgent（先规划后执行）
agents.CreateAgent("plan", "planner", llm, registry)

// 简单对话 → SimpleAgent（无需复杂推理）
agents.CreateAgent("simple", "assistant", llm, registry)
```

---

## 📊 实际案例

### 案例 1：复杂项目分析

```go
mainAgent.Run(`
分析项目架构，生成报告：
1. 使用 Task 工具探索项目结构（agent_type=react, tool_filter=readonly）
2. 使用 Task 工具分析架构设计（agent_type=reflection, tool_filter=readonly）
3. 整合结果，生成报告
`, nil)
```

### 案例 2：成本优化

**配置：**
- 主 Agent：GPT-4（$0.03/1K tokens）
- 子 Agent：DeepSeek（$0.001/1K tokens）

**成本节省：**
```
之前：100% GPT-4 = $30
之后：30% GPT-4 + 70% DeepSeek = $9 + $0.7 = $9.7
节省：68%
```

---

## 📈 性能指标

### 上下文隔离效果

| 场景         | 无隔离（共享历史） | 有隔离（子代理）  |
| ------------ | ------------------ | ----------------- |
| 历史长度     | 100+ 条消息        | 主 20 + 子 10     |
| 上下文清晰度 | 混乱               | 清晰              |
| Token 消耗   | 50,000             | 15,000（节省70%） |

---

## 🔗 相关文档

- [工具响应协议](./tool-response-protocol.md) - ToolFilter 详细说明
- [会话持久化](./session-persistence-guide.md) - 保存子代理会话
- [可观测性](./observability-guide.md) - 追踪子代理执行

---

## ❓ 常见问题

**Q: 子代理会污染主 Agent 的历史吗？**

A: 不会。子代理使用独立历史，执行后自动恢复主 Agent 状态。

**Q: 如何禁用子代理机制？**

A: 设置 `SubagentEnabled = false`：
```go
config.SubagentEnabled = false
```

**Q: 子代理可以访问主 Agent 的工具吗？**

A: 可以，但受工具过滤器限制。

---

**最后更新**: 2026-02-21
