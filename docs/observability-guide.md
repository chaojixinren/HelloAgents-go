# 可观测性指南（Observability）

## 📖 概述

**TraceLogger** 是 HelloAgents-Go 框架的双格式审计轨迹记录器，提供 JSONL（机器可读）和 HTML（人类可读）两种输出格式。

### 核心特性

- ✅ **双格式输出**：JSONL + HTML
- ✅ **流式追加**：实时写入，无需等待会话结束
- ✅ **自动脱敏**：API Key、路径等敏感信息
- ✅ **内置统计**：Token、工具调用、错误统计
- ✅ **可视化界面**：HTML 带交互式面板

---

## 🚀 快速开始

### 1. 自动集成（零配置）

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

// TraceLogger 默认启用
config := core.DefaultConfig()
config.TraceEnabled = true
config.TraceOutputDir = "memory/traces"

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", nil, config, nil, 0, nil)

// 运行任务
agent.Run("分析项目结构", nil)

// 自动生成 trace 文件
// memory/traces/trace-{session_id}.jsonl
// memory/traces/trace-{session_id}.html
```

### 2. 查看 Trace

**JSONL 格式（机器可读）：**
```bash
# 使用 jq 分析
cat memory/traces/trace-xxx.jsonl | jq '.event'

# 过滤工具调用
cat memory/traces/trace-xxx.jsonl | jq 'select(.event=="tool_call")'
```

**HTML 格式（人类可读）：**
```bash
open memory/traces/trace-xxx.html
```

HTML 界面包含：
- 📊 统计面板（Token、工具调用、错误）
- 📝 事件时间线（可折叠）
- 🔍 搜索和过滤
- 🎨 语法高亮

---

## 💡 核心概念

### 事件类型

TraceLogger 记录以下事件：

| 事件类型          | 描述           | 关键字段                    |
| ----------------- | -------------- | --------------------------- |
| `session_start`   | 会话开始       | agent_name, config          |
| `session_end`     | 会话结束       | duration, total_tokens      |
| `step_start`      | ReAct 步骤开始 | step, max_steps             |
| `step_end`        | ReAct 步骤结束 | step, action                |
| `tool_call`       | 工具调用       | tool_name, parameters       |
| `tool_result`     | 工具结果       | tool_name, status, duration |
| `llm_request`     | LLM 请求       | model, messages             |
| `llm_response`    | LLM 响应       | content, usage              |
| `error`           | 错误事件       | error_type, message         |
| `compression`     | 历史压缩       | before_count, after_count   |
| `session_save`    | 会话保存       | filepath                    |
| `circuit_breaker` | 熔断器触发     | tool_name, state            |

### 事件结构

```json
{
  "ts": "2026-02-21T10:30:45.123Z",
  "session_id": "s-20250220-a3f2d8e1",
  "step": 3,
  "event": "tool_call",
  "payload": {
    "tool_name": "Read",
    "parameters": {"path": "config.go"},
    "metadata": {}
  }
}
```

---

## 📝 使用指南

### 1. 手动使用 TraceLogger

```go
import "helloagents-go/hello_agents/observability"

// 创建 logger
logger := observability.NewTraceLogger(
    "memory/traces",  // 输出目录
    true,             // 自动脱敏
    false,            // HTML 不包含原始响应
)

// 记录事件
logger.LogEvent("session_start", map[string]any{
    "agent_name": "MyAgent",
    "config":     config,
}, 0)

logger.LogEvent("tool_call", map[string]any{
    "tool_name":  "Calculator",
    "parameters": map[string]any{"expression": "2+3"},
}, 1)

logger.LogEvent("tool_result", map[string]any{
    "tool_name":   "Calculator",
    "status":      "success",
    "result":      "5",
    "duration_ms": 10,
}, 1)

// 完成会话（生成最终 HTML）
logger.Finalize()
```

### 2. 配置选项

```go
config := core.DefaultConfig()

// 可观测性配置
config.TraceEnabled = true                 // 启用 TraceLogger
config.TraceOutputDir = "memory/traces"    // 输出目录
config.TraceSanitize = true                // 自动脱敏
config.TraceHTMLRawResponse = false        // HTML 包含原始响应
```

### 3. 自动脱敏

TraceLogger 自动脱敏以下信息：

```
// API Key
"api_key": "sk-1234567890abcdef"
// 脱敏后
"api_key": "sk-***"

// Authorization Header
"Authorization": "Bearer token123"
// 脱敏后
"Authorization": "Bearer ***"
```

---

## 📊 实际案例

### 案例 1：问题复盘

**场景：** Agent 执行失败，需要分析原因

```bash
# 1. 查看 HTML trace
open memory/traces/trace-xxx.html

# 2. 定位错误事件
# 在统计面板看到：错误数 = 3

# 3. 查看错误详情
# 点击错误事件，展开详情
# 发现：工具 'MCP' 连续失败 3 次

# 4. 分析根因
# 查看 tool_result 事件
# 错误码：CONNECTION_REFUSED
# 结论：MCP 服务器未启动
```

### 案例 2：性能分析

**场景：** 分析 Token 消耗和工具调用耗时

```bash
# 使用 jq 分析 JSONL
cat memory/traces/trace-xxx.jsonl | jq '
  select(.event=="llm_response") |
  .payload.usage.total_tokens
' | awk '{sum+=$1} END {print "Total tokens:", sum}'

# 分析工具调用耗时
cat memory/traces/trace-xxx.jsonl | jq '
  select(.event=="tool_result") |
  {tool: .payload.tool_name, duration: .payload.duration_ms}
'
```

---

## 🎯 最佳实践

### 1. 生产环境启用 Trace

```go
// ✅ 好：生产环境启用，便于问题排查
config.TraceEnabled = true
config.TraceSanitize = true           // 必须脱敏
config.TraceHTMLRawResponse = false   // 不包含原始响应（节省空间）
```

### 2. 定期清理旧 Trace

```bash
# 删除 7 天前的 trace
find memory/traces -name "trace-*.jsonl" -mtime +7 -delete
find memory/traces -name "trace-*.html" -mtime +7 -delete
```

### 3. 使用 JSONL 进行自动化分析

```go
import (
    "bufio"
    "encoding/json"
    "os"
)

// 读取 JSONL
file, _ := os.Open("memory/traces/trace-xxx.jsonl")
scanner := bufio.NewScanner(file)

toolCalls := map[string]int{}
for scanner.Scan() {
    var event map[string]any
    json.Unmarshal(scanner.Bytes(), &event)
    if event["event"] == "tool_call" {
        payload := event["payload"].(map[string]any)
        name := payload["tool_name"].(string)
        toolCalls[name]++
    }
}

fmt.Println(toolCalls)
// map[Read:5 Write:2 Calculator:1]
```

---

## 🔗 相关文档

- [日志系统](./logging-system-guide.md) - 四种日志范式对比
- [开发日志](./devlog-guide.md) - DevLogTool 使用
- [会话持久化](./session-persistence-guide.md) - 保存和恢复会话

---

## ❓ 常见问题

**Q: TraceLogger 会影响性能吗？**

A: 影响很小：
- JSONL 流式写入，无缓冲
- HTML 增量渲染，实时可查看
- 脱敏操作简单（正则替换）
- 性能开销 < 1%

**Q: 如何禁用 TraceLogger？**

A: 设置 `TraceEnabled = false`：
```go
config.TraceEnabled = false
```

**Q: JSONL 和 HTML 的区别？**

A:
- **JSONL**: 机器可读，适合自动化分析、日志聚合
- **HTML**: 人类可读，适合问题排查、可视化分析

---

**最后更新**: 2026-02-21
