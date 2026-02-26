# 上下文工程指南（Context Engineering）

## 📖 概述

**上下文工程**是 HelloAgents-Go 框架的核心能力，解决长对话中的上下文爆窗、Token 成本爆炸和缓存失效问题。

### 解决的问题

**之前：**
- ❌ 长对话无限增长，最终爆窗
- ❌ 无压缩机制，Token 成本持续增长
- ❌ 工具输出可能塞满上下文
- ❌ 随意修改历史，破坏 KV Cache

**之后：**
- ✅ 自动历史压缩（summary + 最近 N 轮）
- ✅ 缓存友好设计（只追加，不编辑）
- ✅ 工具输出统一截断
- ✅ 支持会话序列化/反序列化

---

## 🚀 快速开始

### 1. 自动历史压缩（简单摘要）

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

// 配置历史压缩（默认：简单摘要）
config := core.DefaultConfig()
config.ContextWindow = 128000          // 上下文窗口大小
config.CompressionThreshold = 0.8     // 压缩阈值（80%）
config.MinRetainRounds = 10           // 保留最近 10 轮
config.EnableSmartCompression = false  // 默认：简单摘要（无需额外 API）

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", nil, config, nil, 0, nil)

// 长对话自动压缩
for i := 0; i < 50; i++ {
    agent.Run(fmt.Sprintf("任务 %d", i), nil)
    // 当历史达到 80% 窗口时，自动压缩为 summary + 最近 10 轮
}
```

**简单摘要示例**：
```
此会话包含 40 轮对话：
- 用户消息：40 条
- 助手消息：40 条
- 总消息数：80 条

（历史已压缩，保留最近 10 轮完整对话）
```

### 2. 工具输出截断

```go
config := core.DefaultConfig()
config.ToolOutputMaxLines = 2000         // 最大行数
config.ToolOutputMaxBytes = 51200        // 最大字节数（50KB）
config.ToolOutputDir = "tool-output"     // 完整输出保存目录
config.ToolOutputTruncateDirection = "head" // 截断方向

agent, _ := agents.NewReActAgent("assistant", llm, "", registry, config, nil, 0, nil)

// 工具输出超过限制时自动截断
agent.Run("读取大文件", nil)
// 自动截断 + 保存完整输出到 tool-output/tool_xxx.json
```

---

## 💡 核心组件

### 1. HistoryManager - 历史管理器

**特性：**
- ✅ 只追加，不编辑（缓存友好）
- ✅ 自动压缩历史
- ✅ 精确的轮次边界检测
- ✅ 支持序列化/反序列化
- ✅ 智能摘要生成（可选）

**使用示例：**
```go
import "helloagents-go/hello_agents/context"

manager := context.NewHistoryManager[core.Message](10, 0.8)

// 添加消息
manager.Append(core.Message{Role: "user", Content: "你好"})
manager.Append(core.Message{Role: "assistant", Content: "你好！"})

// 检查是否需要压缩
if manager.ShouldCompress(128000) {
    // 压缩历史
    manager.Compress(128000, func(msgs []core.Message) string {
        return "历史摘要..."
    })
}

// 获取完整历史（summary + 最近轮次）
messages := manager.GetMessages()
```

### 2. TokenCounter - Token 计数器

**特性：**
- ✅ 本地预估 Token 数（无需 API 调用）
- ✅ 缓存机制（避免重复计算）
- ✅ 增量计算（只计算新增消息）
- ✅ 降级方案（tiktoken 不可用时使用字符估算）

**使用示例：**
```go
import "helloagents-go/hello_agents/context"

counter := context.NewTokenCounter[core.Message]("gpt-4")

// 计算单条消息
tokens := counter.CountMessage(message)

// 计算消息列表
total := counter.CountMessages(messages)

// 缓存统计
stats := counter.GetCacheStats()
// map[string]any{"cached_messages": 50, "total_cached_tokens": 12500}
```

**压缩效果：**
```
压缩前：
- 50 轮对话 = 100 条消息 = 50,000 tokens

压缩后：
- 1 条 summary = 500 tokens
- 最近 10 轮 = 20 条消息 = 10,000 tokens
- 总计：10,500 tokens（节省 79%）
```

### 3. ObservationTruncator - 输出截断器

**特性：**
- ✅ 统一截断规则
- ✅ 多方向截断（head/tail/head_tail）
- ✅ 自动保存完整输出
- ✅ 返回结构化截断信息

**使用示例：**
```go
import "helloagents-go/hello_agents/context"

truncator := context.NewObservationTruncator(2000, 51200, "head", "tool-output")

// 截断长输出
result := truncator.Truncate("search_tool", longOutput)

// 返回结构化信息
// result = map[string]any{
//     "truncated":        true,
//     "preview":          "...",
//     "full_output_path": "tool-output/tool_xxx.json",
//     "stats": map[string]any{
//         "original_lines":  5000,
//         "truncated_lines": 2000,
//     },
// }
```

**截断方向：**
- `head`: 保留开头（适合日志、错误信息）
- `tail`: 保留结尾（适合实时输出）
- `head_tail`: 保留开头和结尾（适合长文件）

---

## 📝 配置选项

### Config 扩展

```go
import "helloagents-go/hello_agents/core"

config := core.DefaultConfig()

// 上下文工程配置
config.ContextWindow = 128000              // 上下文窗口大小
config.CompressionThreshold = 0.8          // 压缩阈值（80%）
config.MinRetainRounds = 10                // 保留最小轮次数
config.EnableSmartCompression = false       // 智能摘要（需额外 LLM 调用）

// 工具输出截断配置
config.ToolOutputMaxLines = 2000           // 最大行数
config.ToolOutputMaxBytes = 51200          // 最大字节数
config.ToolOutputDir = "tool-output"       // 输出目录
config.ToolOutputTruncateDirection = "head" // 截断方向
```

---

## 📊 实际案例

### 案例 1：长对话压缩

**场景：** 50 轮对话，每轮 1000 tokens

**之前：**
```
总 Token: 50 × 1000 = 50,000 tokens
成本: 50,000 × $0.03/1K = $1.50
```

**之后（压缩）：**
```
Summary: 500 tokens
最近 10 轮: 10 × 1000 = 10,000 tokens
总 Token: 10,500 tokens
成本: 10,500 × $0.03/1K = $0.315
节省: 79%
```

### 案例 2：缓存友好设计

**之前（修改历史）：**
```go
// 修改历史中的消息
history[5].Content = "修改后的内容"
// ❌ 破坏 KV Cache，需要重新计算
```

**之后（只追加）：**
```go
// 只追加新消息
manager.Append(core.Message{Role: "summary", Content: "摘要"})
manager.Append(core.Message{Role: "user", Content: "新问题"})
// ✅ 保持缓存有效，节省计算
```

---

## 🎯 最佳实践

### 1. 合理设置压缩阈值

```go
// ❌ 不好：阈值太低，频繁压缩
config.CompressionThreshold = 0.3 // 30% 就压缩

// ✅ 好：阈值适中，平衡性能和成本
config.CompressionThreshold = 0.8 // 80% 时压缩
```

### 2. 保留足够的历史轮次

```go
// ❌ 不好：保留太少，丢失上下文
config.MinRetainRounds = 3

// ✅ 好：保留足够轮次，维持对话连贯性
config.MinRetainRounds = 10
```

### 3. 根据场景选择截断方向

```go
// 日志分析：保留开头（错误通常在开头）
config.ToolOutputTruncateDirection = "head"

// 实时输出：保留结尾（最新信息在结尾）
config.ToolOutputTruncateDirection = "tail"

// 长文件：保留开头和结尾
config.ToolOutputTruncateDirection = "head_tail"
```

---

## 📈 性能指标

### Token 节省效果

| 对话轮次 | 无压缩 Token | 压缩后 Token | 节省比例 |
| -------- | ------------ | ------------ | -------- |
| 10 轮    | 10,000       | 10,000       | 0%       |
| 20 轮    | 20,000       | 11,000       | 45%      |
| 50 轮    | 50,000       | 10,500       | 79%      |
| 100 轮   | 100,000      | 10,500       | 89.5%    |

### 缓存命中率

| 操作类型   | 缓存命中率 | 响应时间 |
| ---------- | ---------- | -------- |
| 修改历史   | 0%         | 2-5 秒   |
| 只追加消息 | 80-95%     | 0.5-1 秒 |

---

## 🔗 相关文档

- [会话持久化](./session-persistence-guide.md) - 保存和恢复会话
- [可观测性](./observability-guide.md) - 追踪上下文使用情况
- [工具响应协议](./tool-response-protocol.md) - 工具输出标准化

---

## ❓ 常见问题

**Q: 压缩会丢失信息吗？**

A: 会丢失部分细节，但保留关键信息：
- 保留：任务目标、重要决策、最近对话
- 丢失：中间步骤的详细过程

**Q: 如何禁用自动压缩？**

A: 设置阈值为 1.0（永不压缩）：
```go
config.CompressionThreshold = 1.0
```

**Q: 工具输出被截断后如何查看完整内容？**

A: 完整输出保存在 `tool-output/` 目录：
```go
result := truncator.Truncate("tool_name", output)
fmt.Println(result["full_output_path"])
// tool-output/tool_20250220_103045.json
```

---

**最后更新**: 2026-02-21
