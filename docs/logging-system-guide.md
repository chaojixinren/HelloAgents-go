# 日志系统指南（Logging System）

## 📖 概述

HelloAgents-Go 框架提供**四种日志范式**，满足不同场景的日志需求：

1. **TraceLogger** - 执行轨迹审计（JSONL + HTML）
2. **AgentLogger** - Agent 运行日志（结构化）
3. **DevLogTool** - 开发日志工具（Agent 可用）
4. **标准 log** - Go 标准日志

---

## 🚀 快速开始

### 1. TraceLogger（执行轨迹）

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/observability"
)

// 启用 TraceLogger
logger := observability.NewTraceLogger("logs", true, false)

config := core.DefaultConfig()
config.TraceEnabled = true

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", nil, config, nil, 0, nil)

// 执行任务
agent.Run("分析项目", nil)

// 查看日志
// - logs/trace.jsonl（机器可读）
// - logs/trace.html（人类可读）
```

### 2. AgentLogger（Agent 日志）

```go
import "helloagents-go/hello_agents/logging"

// 启用 AgentLogger
agentLogger := logging.NewAgentLogger("assistant", "INFO")

// 日志输出：
// [2026-02-21 10:30:45] [INFO] [assistant] Agent 开始执行
// [2026-02-21 10:30:46] [INFO] [assistant] 调用工具: Read
// [2026-02-21 10:30:47] [INFO] [assistant] Agent 完成
```

### 3. DevLogTool（开发日志）

```go
import "helloagents-go/hello_agents/tools/builtin"

// 启用 DevLogTool
config := core.DefaultConfig()
config.DevlogEnabled = true

agent, _ := agents.NewReActAgent("assistant", llm, "", nil, config, nil, 0, nil)

// Agent 可以使用 DevLog 工具
agent.Run("记录开发决策：使用 Redis 作为缓存", nil)
```

### 4. 标准 log

```go
import "log"

// 配置标准 log
log.SetFlags(log.LstdFlags | log.Lshortfile)

agent.Run("分析项目", nil)

// 日志输出：
// 2026/02/21 10:30:45 main.go:15: Agent 开始执行
```

---

## 💡 四种范式对比

| 范式         | 用途           | 格式         | 可读性 | Agent 可用 | 持久化 |
| ------------ | -------------- | ------------ | ------ | ---------- | ------ |
| TraceLogger  | 执行轨迹审计   | JSONL + HTML | 高     | ❌          | ✅      |
| AgentLogger  | Agent 运行日志 | 结构化文本   | 中     | ❌          | ✅      |
| DevLogTool   | 开发决策记录   | JSON         | 高     | ✅          | ✅      |
| 标准 log     | 通用日志       | 文本         | 低     | ❌          | ✅      |

---

## 📝 使用指南

### 1. TraceLogger 详细说明

**特点：**
- ✅ 记录所有 LLM 请求和工具调用
- ✅ 双格式输出（JSONL + HTML）
- ✅ 支持审计和回放

**配置：**
```go
import "helloagents-go/hello_agents/observability"

logger := observability.NewTraceLogger(
    "logs",   // 输出目录
    true,     // 启用脱敏
    false,    // HTML 不包含原始响应
)
```

**查看 HTML 报告：**
```bash
open logs/trace.html
```

### 2. AgentLogger 详细说明

**特点：**
- ✅ 结构化日志（时间戳、级别、消息）
- ✅ 支持多个 Agent 独立日志
- ✅ 可配置日志级别

**配置：**
```go
import "helloagents-go/hello_agents/logging"

logger := logging.NewAgentLogger("assistant", "INFO")
```

**日志级别：**
```go
logger.Debug("调试信息")
logger.Info("普通信息")
logger.Warning("警告信息")
logger.Error("错误信息")
```

**多 Agent 日志：**
```go
logger1 := logging.NewAgentLogger("explorer", "INFO")
logger2 := logging.NewAgentLogger("analyzer", "DEBUG")
```

### 3. DevLogTool 详细说明

**特点：**
- ✅ Agent 可以主动记录日志
- ✅ 7 种日志类别（decision、progress、issue 等）
- ✅ 结构化存储（JSON）

**详细文档：** 参见 [DevLog 指南](./devlog-guide.md)

### 4. 标准 log 详细说明

**特点：**
- ✅ Go 标准库，无需额外依赖
- ✅ 灵活配置
- ✅ 与其他库兼容

**配置：**
```go
import (
    "log"
    "os"
)

// 输出到文件
file, _ := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
log.SetOutput(file)
log.SetFlags(log.LstdFlags | log.Lshortfile)

log.Println("Agent 开始执行")
```

---

## 📊 实际案例

### 案例 1：生产环境监控

```go
// 使用 AgentLogger + 标准 log
agentLogger := logging.NewAgentLogger("production_agent", "INFO")

// 执行任务
result, err := agent.Run("处理用户请求", nil)
if err != nil {
    log.Printf("Agent 执行失败: %v", err)
}
```

### 案例 2：开发调试

```go
// 使用 TraceLogger + AgentLogger
traceLogger := observability.NewTraceLogger("debug_logs", true, true)
agentLogger := logging.NewAgentLogger("debug_agent", "DEBUG")

// 查看日志
// - debug_logs/trace.html（可视化轨迹）
// - 控制台 DEBUG 输出
```

---

## 🎯 最佳实践

### 1. 根据场景选择日志范式

```go
// ✅ 生产环境：AgentLogger + 标准 log
agentLogger := logging.NewAgentLogger("prod", "INFO")

// ✅ 开发调试：TraceLogger + AgentLogger（DEBUG）
traceLogger := observability.NewTraceLogger("debug", true, false)
agentLogger := logging.NewAgentLogger("dev", "DEBUG")

// ✅ 项目管理：DevLogTool
config.DevlogEnabled = true
```

### 2. 日志轮转

```go
import "gopkg.in/natefinch/lumberjack.v2"

log.SetOutput(&lumberjack.Logger{
    Filename:   "agent.log",
    MaxSize:    10, // MB
    MaxBackups: 5,
    MaxAge:     30, // days
})
```

---

## 🔗 相关文档

- [可观测性](./observability-guide.md) - TraceLogger 详细说明
- [DevLog 指南](./devlog-guide.md) - DevLogTool 详细说明

---

## ❓ 常见问题

**Q: 如何同时使用多种日志范式？**

A: 可以组合使用：
```go
traceLogger := observability.NewTraceLogger("logs", true, false)
agentLogger := logging.NewAgentLogger("assistant", "INFO")
config.DevlogEnabled = true
```

**Q: 日志文件太大怎么办？**

A: 使用第三方日志轮转库如 `lumberjack`。

**Q: 如何禁用所有日志？**

A: 将标准 log 输出到 `io.Discard`：
```go
log.SetOutput(io.Discard)
```

---

**最后更新**: 2026-02-21
