# 流式输出与 SSE 指南（Streaming & SSE）

## 📖 概述

**流式输出**是 HelloAgents-Go 框架的实时响应能力，支持 SSE（Server-Sent Events）协议，实现打字机效果和实时进度反馈。

### 核心特性

- ✅ **真正的并发流式**：使用 Go goroutine 和 channel
- ✅ **实时传输**：LLM 生成一个 token 就立即返回
- ✅ **SSE 标准协议**：完美兼容浏览器 EventSource API
- ✅ **8 种事件类型**：AGENT_START、STEP_START、TOOL_CALL、LLM_CHUNK 等

---

## 🚀 快速开始

### 1. 基本流式输出

```go
import (
    "fmt"
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

func main() {
    llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
    agent, _ := agents.NewReActAgent("assistant", llm, "", nil, nil, nil, 0, nil)

    // 流式执行
    eventCh := agent.ArunStream("分析项目结构", nil)
    for event := range eventCh {
        if event.Type == core.StreamEventLLMChunk {
            fmt.Print(event.Data["content"])
        }
    }
}
```

### 2. HTTP SSE 服务端（net/http）

```go
import (
    "encoding/json"
    "fmt"
    "net/http"

    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

func chatStreamHandler(w http.ResponseWriter, r *http.Request) {
    message := r.URL.Query().Get("message")

    llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
    agent, _ := agents.NewReActAgent("assistant", llm, "", nil, nil, nil, 0, nil)

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, _ := w.(http.Flusher)

    eventCh := agent.ArunStream(message, nil)
    for event := range eventCh {
        data, _ := json.Marshal(event.Data)
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
        flusher.Flush()
    }
}

func main() {
    http.HandleFunc("/chat/stream", chatStreamHandler)
    http.ListenAndServe(":8000", nil)
}
```

### 3. 前端 EventSource 客户端

```html
<script>
    function sendMessage() {
        const message = document.getElementById('input').value;
        const output = document.getElementById('output');

        const eventSource = new EventSource(`/chat/stream?message=${message}`);

        eventSource.addEventListener('LLM_CHUNK', (e) => {
            const data = JSON.parse(e.data);
            output.innerHTML += data.content;
        });

        eventSource.addEventListener('AGENT_FINISH', (e) => {
            eventSource.close();
        });
    }
</script>
```

---

## 💡 核心概念

### 8 种流式事件

| 事件类型           | 描述         | 关键字段                  |
| ------------------ | ------------ | ------------------------- |
| `AGENT_START`      | Agent 开始   | input, config             |
| `AGENT_FINISH`     | Agent 结束   | result, duration          |
| `STEP_START`       | 步骤开始     | step, max_steps           |
| `STEP_FINISH`      | 步骤结束     | step, action              |
| `TOOL_CALL_START`  | 工具调用开始 | tool_name, parameters     |
| `TOOL_CALL_FINISH` | 工具调用结束 | tool_name, result, status |
| `LLM_CHUNK`        | LLM 输出块   | content, delta            |
| `THINKING`         | 思考过程     | content                   |
| `ERROR`            | 错误事件     | error_type, message       |

### StreamEvent 数据结构

```go
import "helloagents-go/hello_agents/core"

event := core.StreamEvent{
    Type: core.StreamEventLLMChunk,
    Data: map[string]any{
        "content": "Hello",
        "delta":   "Hello",
    },
    Timestamp: time.Now().Format(time.RFC3339),
    Metadata:  map[string]any{"step": 1},
}

// 转换为 SSE 格式
sseText := event.ToSSE()
// event: LLM_CHUNK
// data: {"content":"Hello","delta":"Hello"}
//
```

---

## 📊 实际案例

### 案例 1：实时代码分析

```go
eventCh := agent.ArunStream("分析项目结构", nil)

fmt.Println("🚀 开始分析项目...")

for event := range eventCh {
    switch event.Type {
    case core.StreamEventStepStart:
        fmt.Printf("\n📍 步骤 %d\n", event.Data["step"])
    case core.StreamEventToolCallStart:
        fmt.Printf("  🔧 %s...", event.Data["tool_name"])
    case core.StreamEventToolCallFinish:
        fmt.Print(" ✅\n")
    case core.StreamEventLLMChunk:
        fmt.Print(event.Data["content"])
    case core.StreamEventAgentFinish:
        fmt.Println("\n\n🎉 分析完成！")
    }
}
```

### 案例 2：多用户并发

```go
import "sync"

// 为每个用户创建独立的 Agent（利用 Go 并发优势）
var userAgents sync.Map

func chatStreamHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    message := r.URL.Query().Get("message")

    agentVal, _ := userAgents.LoadOrStore(userID, createNewAgent())
    agent := agentVal.(*agents.ReActAgent)

    // 每个用户的流式请求在独立 goroutine 中处理
    eventCh := agent.ArunStream(message, nil)
    for event := range eventCh {
        data, _ := json.Marshal(event.Data)
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
    }
}
```

---

## 🎯 最佳实践

### 1. 错误处理

```go
eventCh := agent.ArunStream(message, nil)
for event := range eventCh {
    if event.Type == core.StreamEventError {
        fmt.Fprintf(w, "event: ERROR\ndata: %s\n\n", event.Data["error"])
        return
    }
    // 正常处理...
}
```

### 2. 自定义事件过滤

```go
eventCh := agent.ArunStream(message, nil)
for event := range eventCh {
    // 只发送 LLM 输出和工具调用
    switch event.Type {
    case core.StreamEventLLMChunk,
         core.StreamEventToolCallStart,
         core.StreamEventToolCallFinish:
        data, _ := json.Marshal(event.Data)
        fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
    }
}
```

---

## 📈 性能指标

### 延迟对比

| 模式     | 首字延迟  | 总延迟 | 用户体验 |
| -------- | --------- | ------ | -------- |
| 非流式   | 5-10s     | 5-10s  | 等待     |
| 流式输出 | 200-500ms | 5-10s  | 实时     |

### 资源消耗

| 指标       | 非流式 | 流式 |
| ---------- | ------ | ---- |
| 内存占用   | 高     | 低   |
| 网络带宽   | 突发   | 平稳 |
| 服务器并发 | 低     | 高   |

---

## 🔗 相关文档

- [异步 Agent](./async-agent-guide.md) - ArunStream() 详细说明
- [可观测性](./observability-guide.md) - 追踪流式执行
- [Function Calling](./function-calling-architecture.md) - 流式工具调用

---

## ❓ 常见问题

**Q: SSE 和 WebSocket 的区别？**

A:
- **SSE**: 单向通信（服务端 → 客户端），自动重连，简单易用
- **WebSocket**: 双向通信，需要手动管理连接，更复杂

**Q: Go 的 channel 实现相比 Python async for 有什么优势？**

A: Go 的 channel 是语言级并发原语，结合 goroutine，天然支持高并发。无需 async/await 语法，代码更简洁，且具有更好的并发性能。

**Q: 流式输出的延迟？**

A: 几乎无延迟：LLM 生成 token → 立即通过 channel 发送 → 网络传输 < 10ms

---

**最后更新**: 2026-02-21
