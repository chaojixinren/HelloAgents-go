# 熔断器机制使用指南

> 防止工具连续失败导致的死循环和 Token 浪费

---

## 📖 什么是熔断器？

熔断器（Circuit Breaker）是一种保护机制，当工具连续失败达到阈值时，自动禁用该工具一段时间，避免：

- **死循环**：模型在坏工具上无限重试
- **Token 浪费**：每次失败消耗 200+ tokens，100 次 = 20K tokens
- **资源占用**：持续调用失败的外部 API
- **用户体验差**：任务卡住，无法判断是问题还是正常等待

---

## 🎯 核心特性

### 1. 自动熔断

连续失败 3 次（默认）后，工具自动被禁用：

```
调用 1 → 失败 ❌ (失败计数: 1)
调用 2 → 失败 ❌ (失败计数: 2)
调用 3 → 失败 ❌ (失败计数: 3)
🔴 工具已熔断
调用 4 → 返回 CIRCUIT_OPEN 错误（工具未被实际调用）
```

### 2. 自动恢复

熔断 5 分钟（默认）后，工具自动恢复可用。

### 3. 成功重置

任何一次成功调用会重置失败计数。

### 4. 独立管理

每个工具独立计数，互不影响。

---

## 🚀 快速开始

### 零配置使用（推荐）

框架默认启用熔断器，无需任何配置：

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
)

llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)

// 创建工具注册表（默认启用熔断器）
registry := tools.NewToolRegistry(nil)
registry.RegisterTool(yourTool, false)

// 创建 Agent
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 运行（熔断器自动工作）
result, _ := agent.Run("帮我完成任务", nil)
```

---

## ⚙️ 自定义配置

### 方式 1：通过 Config 配置

```go
config := core.DefaultConfig()
config.CircuitEnabled = true              // 启用熔断器
config.CircuitFailureThreshold = 5        // 5 次失败后熔断（默认 3）
config.CircuitRecoveryTimeout = 600       // 10 分钟后恢复（默认 300）

// 创建熔断器
cb := tools.NewCircuitBreaker(
    config.CircuitFailureThreshold,
    config.CircuitRecoveryTimeout,
    config.CircuitEnabled,
)

// 创建工具注册表
registry := tools.NewToolRegistry(cb)
```

### 方式 2：直接创建熔断器

```go
// 自定义熔断器
cb := tools.NewCircuitBreaker(10, 1800, true) // 10 次失败，30 分钟恢复
registry := tools.NewToolRegistry(cb)
```

### 方式 3：禁用熔断器

```go
cb := tools.NewCircuitBreaker(3, 300, false) // enabled = false
registry := tools.NewToolRegistry(cb)
```

---

## 🔧 手动控制

### 查看工具状态

```go
// 查看单个工具状态
status := registry.CircuitBreaker.GetStatus("tool_name")
fmt.Println(status)
// map[string]any{
//     "state":              "open",
//     "failure_count":      3,
//     "open_since":         1738245123.45,
//     "recover_in_seconds": 245,
// }

// 查看所有工具状态
allStatus := registry.CircuitBreaker.GetAllStatus()
for name, s := range allStatus {
    fmt.Printf("%s: %s (失败 %d 次)\n", name, s["state"], s["failure_count"])
}
```

### 手动开启/关闭熔断

```go
// 手动熔断某个工具
registry.CircuitBreaker.Open("problematic_tool")

// 手动恢复某个工具
registry.CircuitBreaker.Close("problematic_tool")
```

---

## 🎨 工作原理

### 状态机

```
┌─────────────────┐
│  Closed (正常)  │
│  失败计数: 0    │
└────────┬────────┘
         │
         │ 连续失败 >= 3 次
         ▼
┌─────────────────┐
│   Open (熔断)   │
│  拒绝所有调用   │
└────────┬────────┘
         │
         │ 超过 5 分钟
         ▼
┌─────────────────┐
│  Closed (恢复)  │
│  失败计数: 0    │
└─────────────────┘
```

### 错误判断

熔断器基于 `ToolResponse.Status` 判断错误：

```go
// 工具返回
response := tools.Error("执行失败", tools.ErrExecutionError, nil)

// 熔断器判断
if response.Status == tools.StatusError {
    failureCount++ // 增加失败计数
}
```

---

## 📊 实际案例

### 案例 1：MCP 服务器宕机

**之前**：无限重试，浪费大量 Token

**之后**：3 次失败后熔断，5 分钟后自动恢复

**收益**：节省 97% Token

### 案例 2：外部 API 限流

**之前**：持续调用，持续失败，占用请求配额

**之后**：3 次失败后熔断，5 分钟后恢复（API 限流通常也恢复了）

---

## ❓ 常见问题

### Q1: 熔断器会影响性能吗？

**A**: 几乎没有影响。熔断器只在工具执行前后做简单的状态检查，开销可忽略不计。

### Q2: 熔断后 Agent 会怎么做？

**A**: Agent 会收到明确的 `CIRCUIT_OPEN` 错误，可以：
- 尝试其他工具
- 告知用户工具不可用
- 等待恢复后重试

### Q3: 可以针对不同工具设置不同阈值吗？

**A**: 当前版本所有工具共享同一个熔断器配置。如需不同配置，可以创建多个 ToolRegistry。

---

## 🔗 相关文档

- [工具响应协议](./tool-response-protocol.md)
- [可观测性系统](./observability-guide.md)
- [文件操作工具](./file_tools.md)

---

**最后更新**：2026-02-21
**维护者**：HelloAgents 开发团队
