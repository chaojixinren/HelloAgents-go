# 异步 Agent 指南（Async Agent）

## 📖 概述

**异步 Agent** 是 HelloAgents-Go 框架的异步执行能力，利用 Go 的 goroutine 和 channel 机制，支持 `Arun()` 和 `ArunStream()` 方法，实现并行工具调用和流式输出。

### 核心特性

- ✅ **向后兼容**：现有 `Run()` 方法完全不变
- ✅ **工具并行**：用户工具通过 goroutine 并行执行，内置工具串行
- ✅ **生命周期钩子**：OnStart、OnStep、OnToolCall、OnFinish、OnError
- ✅ **流式输出**：实时返回 LLM 输出和工具调用

---

## 🚀 快速开始

### 1. 异步执行

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
)

func main() {
    llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
    agent, _ := agents.NewReActAgent("assistant", llm, "", nil, nil, nil, 0, nil)

    // 异步执行
    hooks := &core.LifecycleHooks{}
    result, err := agent.Arun("分析项目结构", hooks, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

### 2. 流式输出

```go
func main() {
    llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
    agent, _ := agents.NewReActAgent("assistant", llm, "", nil, nil, nil, 0, nil)

    // 流式执行
    eventCh := agent.ArunStream("分析项目结构", nil)
    for event := range eventCh {
        switch event.Type {
        case core.StreamEventLLMChunk:
            fmt.Print(event.Data["content"])
        case core.StreamEventToolCallStart:
            fmt.Printf("\n🔧 调用工具: %s\n", event.Data["tool_name"])
        case core.StreamEventToolCallFinish:
            fmt.Printf("✅ 工具完成: %s\n", event.Data["tool_name"])
        }
    }
}
```

---

## 💡 核心概念

### 1. 异步方法

| 方法            | 同步版本 | 功能                       |
| --------------- | -------- | -------------------------- |
| `Arun()`        | `Run()`  | 异步执行，返回结果         |
| `ArunStream()`  | 无       | 流式执行，返回事件 channel |

### 2. 生命周期钩子

```go
import "helloagents-go/hello_agents/core"

hooks := &core.LifecycleHooks{
    OnStart: func(event core.AgentEvent) {
        fmt.Printf("Agent 开始: %s\n", event.Data["input"])
    },
    OnStep: func(event core.AgentEvent) {
        fmt.Printf("步骤 %d\n", event.Data["step"])
    },
    OnToolCall: func(event core.AgentEvent) {
        fmt.Printf("调用工具: %s\n", event.Data["tool_name"])
    },
    OnFinish: func(event core.AgentEvent) {
        fmt.Printf("Agent 完成: %s\n", event.Data["result"])
    },
    OnError: func(event core.AgentEvent) {
        fmt.Printf("错误: %v\n", event.Data["error"])
    },
}

// 使用钩子
result, _ := agent.Arun("分析项目", hooks, nil)
```

### 3. 工具并行执行

**ReActAgent 并行策略：**
- ✅ **用户工具**：通过 goroutine 并行执行（Read、Write、Search 等）
- ✅ **内置工具**：串行执行（Thought、Finish）

```go
// Agent 会通过 goroutine 并行调用 Read、Search、Calculator
result, _ := agent.Arun("读取 config.go，搜索文档，计算 2+3", hooks, nil)
// 执行时间：max(Read, Search, Calculator) 而非 sum
```

---

## 📝 使用指南

### 1. 基本异步执行

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
    "helloagents-go/hello_agents/tools/builtin"
)

// 创建 Agent
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(builtin.NewReadTool("./", registry), false)

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 异步执行
result, _ := agent.Arun("读取 README.md 并搜索相关文档", nil, nil)
fmt.Println(result)
```

### 2. 流式输出

```go
eventCh := agent.ArunStream("分析项目", nil)

for event := range eventCh {
    switch event.Type {
    case core.StreamEventAgentStart:
        fmt.Println("🚀 Agent 开始")
    case core.StreamEventStepStart:
        fmt.Printf("\n📍 步骤 %d\n", event.Data["step"])
    case core.StreamEventThinking:
        fmt.Printf("💭 思考: %s\n", event.Data["content"])
    case core.StreamEventToolCallStart:
        fmt.Printf("🔧 调用: %s\n", event.Data["tool_name"])
    case core.StreamEventToolCallFinish:
        fmt.Printf("✅ 完成: %s\n", event.Data["tool_name"])
    case core.StreamEventLLMChunk:
        fmt.Print(event.Data["content"])
    case core.StreamEventAgentFinish:
        fmt.Println("\n🎉 Agent 完成")
    }
}
```

### 3. 生命周期钩子

```go
// 日志钩子
loggingHooks := &core.LifecycleHooks{
    OnStart: func(event core.AgentEvent) {
        fmt.Printf("[START] 输入: %s\n", event.Data["input"])
    },
    OnToolCall: func(event core.AgentEvent) {
        fmt.Printf("[TOOL] %s: %v\n", event.Data["tool_name"], event.Data["parameters"])
    },
    OnFinish: func(event core.AgentEvent) {
        fmt.Printf("[FINISH] 完成\n")
    },
}

result, _ := agent.Arun("分析项目", loggingHooks, nil)
```

---

## 📊 实际案例

### 案例 1：并行工具调用

**场景：** 同时读取多个文件

```go
// Agent 会通过 goroutine 并行读取 3 个文件
result, _ := agent.Arun(`
    读取以下文件：
    1. config.go
    2. main.go
    3. utils.go
`, nil, nil)

// 执行时间：max(read1, read2, read3) 而非 sum
```

**性能提升：**
```
串行执行：3 × 1s = 3s
并行执行：max(1s, 1s, 1s) = 1s
提升：3 倍
```

### 案例 2：实时进度显示

```go
eventCh := agent.ArunStream("分析项目结构", nil)

fmt.Println("🚀 开始分析项目...")

for event := range eventCh {
    switch event.Type {
    case core.StreamEventStepStart:
        fmt.Printf("\n📍 步骤 %d/%d\n", event.Data["step"], event.Data["max_steps"])
    case core.StreamEventToolCallStart:
        fmt.Printf("  🔧 %s...", event.Data["tool_name"])
    case core.StreamEventToolCallFinish:
        fmt.Printf(" ✅ (%dms)\n", event.Data["duration_ms"])
    case core.StreamEventAgentFinish:
        fmt.Println("\n🎉 分析完成！")
    }
}
```

---

## 🎯 最佳实践

### 1. 使用 goroutine 批量处理

```go
// 并行执行多个独立任务
var wg sync.WaitGroup
results := make([]string, len(tasks))

for i, task := range tasks {
    wg.Add(1)
    go func(idx int, t string) {
        defer wg.Done()
        result, _ := agent.Arun(t, nil, nil)
        results[idx] = result
    }(i, task)
}

wg.Wait()
```

### 2. 超时控制

```go
import "context"

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

// 使用 context 控制超时
resultCh := make(chan string, 1)
go func() {
    result, _ := agent.Arun("长时间任务", nil, nil)
    resultCh <- result
}()

select {
case result := <-resultCh:
    fmt.Println(result)
case <-ctx.Done():
    fmt.Println("任务超时")
}
```

---

## 🔗 相关文档

- [流式输出](./streaming-sse-guide.md) - SSE 协议和前端集成
- [可观测性](./observability-guide.md) - 追踪异步执行
- [Function Calling](./function-calling-architecture.md) - 异步工具调用

---

## ❓ 常见问题

**Q: Run() 和 Arun() 可以混用吗？**

A: 可以，`Run()` 是同步版本，`Arun()` 支持生命周期钩子。

**Q: 流式输出的性能开销？**

A: 几乎没有开销，使用 Go 原生 channel 机制，内存占用低。

**Q: 如何禁用工具并行执行？**

A: 当前所有用户工具默认并行，可通过自定义钩子控制串行执行。

---

**最后更新**: 2026-02-21
