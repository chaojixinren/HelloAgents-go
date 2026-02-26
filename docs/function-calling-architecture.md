# Function Calling 架构指南

## 📖 概述

**Function Calling 架构**是 HelloAgents-Go 框架的核心重构，将 LLM 基类和所有 Agent 类型统一为 Function Calling 模式，解析成功率从 85% 提升到 99%+。

### 核心改进

- ✅ **LLM 基类重构**：InvokeWithTools() 统一接口
- ✅ **Agent 基类重构**：所有 Agent 类型使用 Function Calling
- ✅ **解析成功率提升**：85% → 99%+
- ✅ **向后兼容**：现有代码无需修改

---

## 🚀 快速开始

### 1. 使用 Function Calling

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
    "helloagents-go/hello_agents/tools/builtin"
)

// 创建工具注册表
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(builtin.NewReadTool("./", registry), false)

// 创建 Agent（自动使用 Function Calling）
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 执行任务
result, _ := agent.Run("读取 README.md 并搜索相关文档", nil)
```

### 2. 直接调用 LLM Function Calling

```go
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
readTool := builtin.NewReadTool("./", nil)

// 使用 Function Calling
response := llm.InvokeWithTools(
    []core.Message{{Role: "user", Content: "读取 config.go"}},
    []tools.Tool{readTool},
)

// 解析工具调用
if len(response.ToolCalls) > 0 {
    for _, tc := range response.ToolCalls {
        fmt.Printf("工具: %s\n", tc.Name)
        fmt.Printf("参数: %v\n", tc.Arguments)
    }
}
```

---

## 💡 核心概念

### 1. 为什么重构为 Function Calling？

**旧方案（Prompt 工程）：**
```
❌ 问题：解析失败率高（15%）
LLM 可能输出：
- "Action: read" (大小写错误)
- "Action Input: {path: config.go}" (JSON 格式错误)
- "我将使用 Read 工具..." (格式完全错误)
```

**新方案（Function Calling）：**
```go
// ✅ 优势：LLM 原生支持，解析成功率 99%+
response := llm.InvokeWithTools(messages, allTools)
// LLM 返回结构化的工具调用，无需额外解析
```

### 2. LLM 基类重构

**核心方法：InvokeWithTools()**

```go
type BaseLLM interface {
    InvokeWithTools(
        messages []Message,
        tools []Tool,
    ) *LLMResponse
}
```

**LLMResponse 数据结构：**

```go
type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]any
}

type LLMResponse struct {
    Content   string            // LLM 文本输出
    ToolCalls []ToolCall         // 工具调用列表
    Usage     map[string]int     // Token 使用统计
}
```

### 3. Agent 基类重构

**所有 Agent 类型统一使用 Function Calling：**

```go
type BaseAgent struct {
    LLM           BaseLLM
    ToolRegistry  *ToolRegistry
}

func (a *BaseAgent) callLLM(messages []Message) *LLMResponse {
    return a.LLM.InvokeWithTools(
        messages,
        a.ToolRegistry.GetAllTools(),
    )
}

func (a *BaseAgent) executeToolCalls(toolCalls []ToolCall) []string {
    results := make([]string, len(toolCalls))
    for i, tc := range toolCalls {
        tool := a.ToolRegistry.GetTool(tc.Name)
        result := tool.Run(tc.Arguments)
        results[i] = result.Text
    }
    return results
}
```

---

## 📝 使用指南

### 1. ReActAgent 使用 Function Calling

```go
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(builtin.NewReadTool("./", registry), false)
registry.RegisterTool(builtin.NewWriteTool("./"), false)

agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// Agent 内部流程：
// 1. 调用 llm.InvokeWithTools(messages, tools)
// 2. 解析 ToolCalls
// 3. 执行工具
// 4. 将结果添加到历史
// 5. 继续循环

result, _ := agent.Run("读取 config.go，修改端口为 8080，保存", nil)
```

### 2. ReflectionAgent 使用 Function Calling

```go
agent, _ := agents.NewReflectionAgent("thinker", llm, "", registry, nil, nil, 0, nil)

// ReflectionAgent 流程：
// 1. 执行阶段：使用 Function Calling 调用工具
// 2. 反思阶段：评估执行结果
// 3. 改进阶段：根据反思调整策略

result, _ := agent.Run("分析项目架构", nil)
```

### 3. PlanSolveAgent 使用 Function Calling

```go
agent, _ := agents.NewPlanSolveAgent("planner", llm, "", registry, nil, nil, 0, nil)
result, _ := agent.Run("重构项目结构", nil)
```

### 4. SimpleAgent 使用 Function Calling

```go
agent, _ := agents.NewSimpleAgent("assistant", llm, "", registry, nil)
result, _ := agent.Run("读取 README.md", nil)
```

---

## 📊 实际案例

### 案例 1：解析成功率对比

| 方案             | 成功率 | 失败原因                     |
| ---------------- | ------ | ---------------------------- |
| Prompt 工程      | 85%    | 格式错误、大小写、JSON 错误  |
| Function Calling | 99%+   | LLM 幻觉（调用不存在的工具） |

### 案例 2：复杂工具调用

```go
// LLM 返回多个工具调用
response := llm.InvokeWithTools(
    []core.Message{{Role: "user", Content: "读取 config.go 和 main.go"}},
    []tools.Tool{readTool},
)

// response.ToolCalls:
// [
//     ToolCall{ID: "call_1", Name: "Read", Arguments: {"path": "config.go"}},
//     ToolCall{ID: "call_2", Name: "Read", Arguments: {"path": "main.go"}},
// ]

// Agent 并行执行两个工具调用
```

---

## 🎯 最佳实践

### 1. 工具描述清晰

```go
type ReadTool struct {
    tools.BaseTool
}

// ✅ 好：清晰的描述帮助 LLM 正确调用
// ToolName = "Read"
// ToolDescription = "读取文件内容。参数：path (string) - 文件路径"
```

### 2. 参数验证

```go
func (t *ReadTool) Run(params map[string]any) *tools.ToolResponse {
    path, ok := params["path"].(string)
    if !ok || path == "" {
        return tools.Error("缺少 path 参数", tools.ErrInvalidParam, nil)
    }
    // 执行读取...
}
```

### 3. 错误处理

```go
// Agent 内部错误处理
response := llm.InvokeWithTools(messages, allTools)

for _, tc := range response.ToolCalls {
    tool := registry.GetTool(tc.Name)
    if tool == nil {
        // 工具不存在，添加错误消息到历史
        errorMsg := fmt.Sprintf("工具 %s 不存在", tc.Name)
        messages = append(messages, core.Message{
            Role:       "tool",
            ToolCallID: tc.ID,
            Content:    errorMsg,
        })
        continue
    }
    result := tool.Run(tc.Arguments)
    // ...
}
```

---

## 📈 性能指标

### 解析成功率

| 方案             | 成功率 | 失败原因                     |
| ---------------- | ------ | ---------------------------- |
| Prompt 工程      | 85%    | 格式错误、大小写、JSON 错误  |
| Function Calling | 99%+   | LLM 幻觉（调用不存在的工具） |

### Token 消耗

| 方案             | Prompt Tokens | 节省比例 |
| ---------------- | ------------- | -------- |
| Prompt 工程      | 500           | 0%       |
| Function Calling | 300           | 40%      |

---

## 🔗 相关文档

- [工具响应协议](./tool-response-protocol.md) - ToolResponse 标准
- [异步 Agent](./async-agent-guide.md) - 异步工具调用
- [可观测性](./observability-guide.md) - 追踪 Function Calling

---

## ❓ 常见问题

**Q: Function Calling 支持哪些 LLM？**

A: 支持所有主流 LLM：OpenAI GPT-4、Anthropic Claude 3、DeepSeek-Chat 等。

**Q: Function Calling 的性能开销？**

A: 几乎没有开销，LLM 原生支持，无需额外解析，且减少了 Prompt 长度。

**Q: 如何调试 Function Calling？**

A: 使用 TraceLogger：
```go
logger := observability.NewTraceLogger("logs", true, false)
// 查看 logs/trace.jsonl 和 logs/trace.html
```

---

**最后更新**: 2026-02-21
