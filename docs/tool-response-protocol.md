# 工具响应协议（ToolResponse Protocol）

## 📖 概述

**ToolResponse 协议**是 HelloAgents-Go 框架的标准化工具响应格式，解决了传统字符串返回的模糊性问题。

### 解决的问题

**之前（字符串返回）：**
```go
func (t *MyTool) Run(params map[string]any) string {
    return "计算结果: 5" // 无法区分成功/失败/部分成功
}
```

**问题：**
- ❌ 状态不明确（成功？失败？）
- ❌ 错误信息难以解析（需要正则匹配）
- ❌ 无法携带结构化数据
- ❌ Agent 需要"猜测"工具执行结果

**之后（ToolResponse 协议）：**
```go
func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    return tools.Success(
        "计算结果: 5",
        map[string]any{"result": 5, "expression": "2+3"},
    )
}
```

**优势：**
- ✅ 状态明确（SUCCESS/PARTIAL/ERROR）
- ✅ 标准错误码（15种）
- ✅ 结构化数据载荷
- ✅ Agent 直接读取 Status 字段

---

## 🚀 快速开始

### 1. 创建成功响应

```go
import "helloagents-go/hello_agents/tools"

// 简单成功响应
response := tools.Success(
    "文件读取成功",
    map[string]any{"content": "Hello World", "size": 11},
)

// 带统计信息
response := tools.SuccessWithStats(
    "搜索完成，找到 3 条结果",
    map[string]any{"results": results},
    map[string]any{"time_ms": 245, "count": 3},
)
```

### 2. 创建错误响应

```go
import "helloagents-go/hello_agents/tools"

// 文件不存在
response := tools.Error(
    "文件 'config.go' 不存在",
    tools.ErrNotFound,
    nil,
)

// 参数无效
response := tools.Error(
    "参数 'path' 不能为空",
    tools.ErrInvalidParam,
    nil,
)
```

### 3. 创建部分成功响应

```go
// 结果被截断
response := tools.Partial(
    "搜索结果（前 100 条）",
    map[string]any{"results": results[:100], "total": 500},
    "结果过多，已截断",
)
```

---

## 💡 核心概念

### 三种状态

| 状态      | 含义                   | 使用场景                       |
| --------- | ---------------------- | ------------------------------ |
| `SUCCESS` | 任务完全按预期执行     | 正常完成                       |
| `PARTIAL` | 结果可用但存在折扣     | 截断、回退、部分失败           |
| `ERROR`   | 无有效结果（致命错误） | 文件不存在、权限错误、执行失败 |

### 标准错误码（15种）

```go
import "helloagents-go/hello_agents/tools"

// 资源相关
tools.ErrNotFound          // 资源不存在
tools.ErrAlreadyExists     // 资源已存在
tools.ErrPermissionDenied  // 权限不足

// 参数相关
tools.ErrInvalidParam      // 参数无效
tools.ErrInvalidFormat     // 格式错误

// 执行相关
tools.ErrExecutionError    // 执行错误
tools.ErrTimeout           // 超时
tools.ErrConflict          // 冲突（乐观锁）

// 系统相关
tools.ErrCircuitOpen       // 熔断器开启
tools.ErrRateLimit         // 速率限制
tools.ErrNetworkError      // 网络错误
tools.ErrServiceUnavailable // 服务不可用

// 其他
tools.ErrPartialSuccess    // 部分成功
tools.ErrDeprecated        // 已弃用
tools.ErrUnknown           // 未知错误
```

### ToolResponse 数据结构

```go
type ToolResponse struct {
    Status    ToolStatus        // SUCCESS / PARTIAL / ERROR
    Text      string            // 给 LLM 阅读的格式化文本
    Data      map[string]any    // 结构化数据载荷
    ErrorInfo map[string]any    // 错误信息（仅 ERROR 时）
    Stats     map[string]any    // 运行统计（时间、token等）
    Context   map[string]any    // 上下文信息（参数、环境等）
}
```

---

## 📝 使用指南

### 实现自定义工具

```go
package main

import (
    "helloagents-go/hello_agents/tools"
)

type MyTool struct {
    tools.BaseTool
}

func NewMyTool() *MyTool {
    t := &MyTool{}
    t.ToolName = "MyTool"
    t.ToolDescription = "我的自定义工具"
    return t
}

func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    // 1. 参数验证
    input, ok := params["input"].(string)
    if !ok || input == "" {
        return tools.Error(
            "参数 'input' 不能为空",
            tools.ErrInvalidParam,
            nil,
        )
    }

    // 2. 执行业务逻辑
    result := doWork(input)

    // 3. 返回成功响应
    return tools.Success(
        fmt.Sprintf("处理完成: %s", result),
        map[string]any{"result": result},
    )
}

func (t *MyTool) GetParameters() []tools.ToolParameter {
    return []tools.ToolParameter{
        {
            Name:        "input",
            Type:        "string",
            Description: "输入内容",
            Required:    true,
        },
    }
}
```

### 在 Agent 中使用

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
)

// 注册工具
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(NewMyTool(), false)

// 创建 Agent
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// Agent 自动处理 ToolResponse
result, _ := agent.Run("使用 MyTool 处理数据", nil)
```

**Agent 内部处理逻辑：**
```go
// Agent 执行工具
toolResponse := registry.ExecuteTool("MyTool", params)

// 根据状态处理
switch toolResponse.Status {
case tools.StatusSuccess:
    // 成功：继续执行
    fmt.Printf("✅ %s\n", toolResponse.Text)

case tools.StatusPartial:
    // 部分成功：提示 Agent 注意
    fmt.Printf("⚠️ %s\n", toolResponse.Text)

case tools.StatusError:
    // 错误：明确提示错误码和信息
    code := toolResponse.ErrorInfo["code"]
    fmt.Printf("❌ 错误 [%s]: %s\n", code, toolResponse.Text)
}
```

---

## 🔄 迁移指南

### 旧工具（字符串返回）

```go
type OldTool struct {
    tools.BaseTool
}

func (t *OldTool) Run(params map[string]any) string {
    path, _ := params["path"].(string)
    if path == "" {
        return "错误: 参数 'path' 不能为空"
    }
    content, err := os.ReadFile(path)
    if err != nil {
        return "错误: 文件不存在"
    }
    return fmt.Sprintf("文件内容: %s", content)
}
```

### 新工具（ToolResponse 协议）

```go
type NewTool struct {
    tools.BaseTool
}

func (t *NewTool) Run(params map[string]any) *tools.ToolResponse {
    // 参数验证
    path, _ := params["path"].(string)
    if path == "" {
        return tools.Error("参数 'path' 不能为空", tools.ErrInvalidParam, nil)
    }

    // 执行逻辑
    content, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return tools.Error(
                fmt.Sprintf("文件 '%s' 不存在", path),
                tools.ErrNotFound, nil,
            )
        }
        return tools.Error(err.Error(), tools.ErrExecutionError, nil)
    }

    return tools.Success(
        "文件读取成功",
        map[string]any{"content": string(content), "path": path},
    )
}
```

**迁移步骤：**
1. 修改返回类型：`string` → `*tools.ToolResponse`
2. 成功时使用 `tools.Success()`
3. 错误时使用 `tools.Error()` + 标准错误码
4. 部分成功使用 `tools.Partial()`

---

## 📊 实际案例

### 案例 1：文件读取工具

```go
import "helloagents-go/hello_agents/tools/builtin"

readTool := builtin.NewReadTool("./", registry)

// 成功读取
response := readTool.Run(map[string]any{"path": "config.go"})
// ToolResponse{
//     Status: SUCCESS,
//     Text:   "文件读取成功",
//     Data:   {"content": "...", "size": 1024},
// }

// 文件不存在
response := readTool.Run(map[string]any{"path": "not_exist.go"})
// ToolResponse{
//     Status:    ERROR,
//     Text:      "文件 'not_exist.go' 不存在",
//     ErrorInfo: {"code": "NOT_FOUND", "message": "..."},
// }
```

### 案例 2：计算器工具

```go
import "helloagents-go/hello_agents/tools/builtin"

calc := builtin.NewCalculatorTool()

// 成功计算
response := calc.Run(map[string]any{"expression": "2 + 3"})
// ToolResponse{
//     Status: SUCCESS,
//     Text:   "计算结果: 5",
//     Data:   {"result": 5, "expression": "2+3"},
// }

// 语法错误
response := calc.Run(map[string]any{"expression": "2 +"})
// ToolResponse{
//     Status:    ERROR,
//     Text:      "表达式语法错误",
//     ErrorInfo: {"code": "INVALID_FORMAT", "message": "..."},
// }
```

### 案例 3：搜索工具（部分成功）

```go
// 结果过多，自动截断
response := searchTool.Run(map[string]any{"query": "golang"})
// ToolResponse{
//     Status: PARTIAL,
//     Text:   "搜索完成（前 100 条结果）",
//     Data:   {"results": [...], "total": 500, "truncated": true},
// }
```

---

## 🎯 最佳实践

### 1. 明确的错误码

```go
// ❌ 不好：使用通用错误码
return tools.Error("出错了", tools.ErrUnknown, nil)

// ✅ 好：使用精确的错误码
return tools.Error(
    "无权限访问文件 'secret.txt'",
    tools.ErrPermissionDenied, nil,
)
```

### 2. 丰富的数据载荷

```go
// ❌ 不好：只返回文本
return tools.Success("找到 3 个文件", nil)

// ✅ 好：返回结构化数据
return tools.Success(
    "找到 3 个文件",
    map[string]any{
        "files":     []string{"a.go", "b.go", "c.go"},
        "count":     3,
        "directory": "/src",
    },
)
```

### 3. 有用的统计信息

```go
return tools.SuccessWithStats(
    "搜索完成",
    map[string]any{"results": results},
    map[string]any{
        "time_ms":   245,
        "count":     10,
        "api_calls": 1,
    },
)
```

---

## 🔗 相关文档

- [熔断器机制](./circuit-breaker-guide.md) - 基于 ToolResponse 的错误判断
- [文件工具](./file_tools.md) - ReadTool、WriteTool 使用 ToolResponse
- [可观测性](./observability-guide.md) - TraceLogger 记录 ToolResponse

---

## ❓ 常见问题

**Q: 如何判断工具是否支持新协议？**

A: Go 版本中所有工具统一返回 `*tools.ToolResponse`，通过接口约束保证类型安全：
```go
type Tool interface {
    Run(params map[string]any) *ToolResponse
    GetParameters() []ToolParameter
    // ...
}
```

**Q: PARTIAL 和 ERROR 的区别？**

A:
- `PARTIAL`: 有结果，但不完整（截断、部分失败）
- `ERROR`: 无有效结果（致命错误）

---

**最后更新**: 2026-02-21
