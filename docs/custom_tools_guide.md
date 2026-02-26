# HelloAgents-Go 自定义工具开发指南

> 本指南帮助你快速创建和注册自己的自定义工具，与框架内置工具无缝集成

---

## 📚 目录

- [快速开始](#快速开始)
- [三种实现方式](#三种实现方式)
- [工具模板](#工具模板)
- [实战示例](#实战示例)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 🚀 快速开始

### 安装框架

```bash
go mod download
```

### 最简单的自定义工具

```go
package main

import (
    "fmt"
    "strings"

    "helloagents-go/hello_agents/tools"
)

type MyFirstTool struct {
    tools.BaseTool
}

func NewMyFirstTool() *MyFirstTool {
    t := &MyFirstTool{}
    t.ToolName = "my_first_tool"
    t.ToolDescription = "这是我的第一个自定义工具，用于演示基本用法"
    return t
}

func (t *MyFirstTool) Run(params map[string]any) *tools.ToolResponse {
    input, ok := params["input"].(string)
    if !ok || input == "" {
        return tools.Error(
            "参数 'input' 不能为空",
            tools.ErrInvalidParam,
            nil,
        )
    }

    result := strings.ToUpper(input)

    return tools.Success(
        fmt.Sprintf("处理结果: %s", result),
        map[string]any{"original": input, "processed": result},
    )
}

func (t *MyFirstTool) GetParameters() []tools.ToolParameter {
    return []tools.ToolParameter{
        {
            Name:        "input",
            Type:        "string",
            Description: "要处理的输入文本",
            Required:    true,
        },
    }
}
```

### 注册和使用

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
)

// 1. 创建工具注册表
registry := tools.NewToolRegistry(nil)

// 2. 注册自定义工具（与内置工具完全一致）
registry.RegisterTool(NewMyFirstTool(), false)

// 3. 创建 Agent
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// 4. 使用工具
result, _ := agent.Run("使用 my_first_tool 处理文本 'hello world'", nil)
fmt.Println(result)
```

---

## 🎯 三种实现方式

HelloAgents-Go 提供三种渐进式的工具实现方式，适应不同复杂度的需求：

### 方式 1：函数式工具（最简单）

适合简单的一次性工具，使用函数注册。

```go
// 定义函数
func simpleCalculator(params map[string]any) *tools.ToolResponse {
    a, _ := params["a"].(float64)
    b, _ := params["b"].(float64)
    op, _ := params["operation"].(string)

    var result float64
    switch op {
    case "add":
        result = a + b
    case "sub":
        result = a - b
    case "mul":
        result = a * b
    case "div":
        if b == 0 {
            return tools.Error("除数不能为零", tools.ErrInvalidParam, nil)
        }
        result = a / b
    default:
        return tools.Error("不支持的运算", tools.ErrInvalidParam, nil)
    }

    return tools.Success(
        fmt.Sprintf("计算结果: %g", result),
        map[string]any{"result": result},
    )
}

// 注册函数式工具
registry.RegisterFunction(simpleCalculator, "simple_calc", "执行简单的数学运算")
```

### 方式 2：标准工具类（推荐）

实现 `tools.Tool` 接口，获得完整的工具功能。

```go
type WeatherTool struct {
    tools.BaseTool
    apiKey string
}

func NewWeatherTool(apiKey string) *WeatherTool {
    t := &WeatherTool{apiKey: apiKey}
    t.ToolName = "weather"
    t.ToolDescription = "查询指定城市的天气信息"
    return t
}

func (t *WeatherTool) Run(params map[string]any) *tools.ToolResponse {
    city, ok := params["city"].(string)
    if !ok || city == "" {
        return tools.Error("参数 'city' 不能为空", tools.ErrInvalidParam, nil)
    }

    weatherData := t.fetchWeather(city)
    if weatherData == nil {
        return tools.Error(
            fmt.Sprintf("未找到城市 '%s' 的天气信息", city),
            tools.ErrNotFound, nil,
        )
    }

    return tools.Success(
        fmt.Sprintf("%s 的天气: %s, 温度: %d°C", city, weatherData["description"], weatherData["temp"]),
        weatherData,
    )
}

func (t *WeatherTool) GetParameters() []tools.ToolParameter {
    return []tools.ToolParameter{
        {
            Name:        "city",
            Type:        "string",
            Description: "要查询的城市名称",
            Required:    true,
        },
    }
}

func (t *WeatherTool) fetchWeather(city string) map[string]any {
    // 实际实现中调用真实的天气 API
    return map[string]any{
        "city":        city,
        "description": "晴天",
        "temp":        25,
        "humidity":    60,
    }
}
```

### 方式 3：可展开工具（高级）

将一个工具展开为多个子工具，使用方法名映射。

```go
type DatabaseTool struct {
    tools.BaseTool
    connStr string
}

func NewDatabaseTool(connStr string) *DatabaseTool {
    t := &DatabaseTool{connStr: connStr}
    t.ToolName = "database"
    t.ToolDescription = "数据库操作工具集"
    t.Expandable = true
    return t
}

// 子工具：查询
func (t *DatabaseTool) Query(params map[string]any) *tools.ToolResponse {
    sql, _ := params["sql"].(string)
    results := t.executeQuery(sql)
    return tools.Success(
        fmt.Sprintf("查询成功，返回 %d 行", len(results)),
        map[string]any{"results": results},
    )
}

// 子工具：插入
func (t *DatabaseTool) Insert(params map[string]any) *tools.ToolResponse {
    table, _ := params["table"].(string)
    data, _ := params["data"].(map[string]any)
    rowID := t.executeInsert(table, data)
    return tools.Success(
        fmt.Sprintf("数据插入成功，ID: %d", rowID),
        map[string]any{"inserted_id": rowID},
    )
}

// 注册：自动展开为 database_query 和 database_insert
registry.RegisterTool(NewDatabaseTool("sqlite:///mydb.db"), false)
```

---

## 📝 工具模板

框架提供了三个开箱即用的模板，位于 `example/custom_tools/` 目录：

1. **simple_tool_template/** - 简单工具模板（最小实现）
2. **advanced_tool_template/** - 高级工具模板（完整特性）
3. **expandable_tool_template/** - 可展开工具模板（多功能）

---

## 🎓 实战示例

框架提供了多个真实场景的示例工具，位于 `example/custom_tools/` 目录：

1. **weather_tool/** - 天气查询工具
2. **code_formatter_tool/** - 代码格式化工具
3. **complete_example/** - 完整示例

---

## ✅ 最佳实践

### 1. 错误处理

始终使用标准错误码，提供清晰的错误信息：

```go
// ✅ 好的做法
return tools.Error(
    "参数 'city' 不能为空",
    tools.ErrInvalidParam,
    map[string]any{"provided_params": params},
)

// ❌ 不好的做法
return tools.Error("出错了", tools.ErrUnknown, nil)
```

### 2. 参数验证

在 `Run()` 方法开始时验证所有必需参数：

```go
func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    required := []string{"city", "date"}
    for _, param := range required {
        if _, ok := params[param]; !ok {
            return tools.Error(
                fmt.Sprintf("缺少必需参数: %s", param),
                tools.ErrInvalidParam, nil,
            )
        }
    }
    // 继续执行工具逻辑
}
```

### 3. 结构化数据

返回结构化的 Data 字段：

```go
return tools.Success(
    "查询成功，找到 3 条记录",
    map[string]any{
        "records":       records,
        "count":         3,
        "query_time_ms": 45,
    },
)
```

### 4. 添加日志

```go
import "log"

func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    log.Printf("执行工具 %s，参数: %v", t.ToolName, params)

    result, err := t.doWork(params)
    if err != nil {
        log.Printf("工具执行失败: %v", err)
        return tools.Error(err.Error(), tools.ErrExecutionError, nil)
    }

    log.Println("工具执行成功")
    return tools.Success(result, nil)
}
```

### 5. 并发安全

Go 中尤其需要注意并发安全：

```go
type MyTool struct {
    tools.BaseTool
    mu    sync.Mutex
    cache map[string]any
}

func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    t.mu.Lock()
    defer t.mu.Unlock()
    // 安全地访问共享状态
}
```

---

## ❓ 常见问题

### Q1: 如何在工具中访问 Agent 的上下文？

工具应该是无状态的，不应该直接访问 Agent。如果需要上下文信息，通过参数传递：

```go
// ❌ 不推荐
type MyTool struct {
    agent *agents.ReActAgent // 不要这样做
}

// ✅ 推荐
func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    ctx, _ := params["context"].(map[string]any)
    // 使用传入的上下文
}
```

### Q2: 如何处理长时间运行的任务？

使用 goroutine 或返回 PARTIAL 状态：

```go
func (t *MyTool) Run(params map[string]any) *tools.ToolResponse {
    taskID := t.startBackgroundTask(params)
    return tools.Partial(
        fmt.Sprintf("任务已启动，ID: %s", taskID),
        map[string]any{"task_id": taskID, "status": "running"},
        "后台任务已启动",
    )
}
```

### Q3: 如何测试自定义工具？

编写单元测试：

```go
func TestMyToolSuccess(t *testing.T) {
    tool := NewMyTool()
    response := tool.Run(map[string]any{"input": "test"})

    if response.Status != tools.StatusSuccess {
        t.Errorf("expected success, got %s", response.Status)
    }
    if response.Data["processed"] != "TEST" {
        t.Errorf("unexpected result: %v", response.Data["processed"])
    }
}

func TestMyToolError(t *testing.T) {
    tool := NewMyTool()
    response := tool.Run(map[string]any{}) // 缺少参数

    if response.Status != tools.StatusError {
        t.Errorf("expected error, got %s", response.Status)
    }
}
```

### Q4: 工具可以调用其他工具吗？

可以，通过 ToolRegistry：

```go
type ComposeTool struct {
    tools.BaseTool
    registry *tools.ToolRegistry
}

func (t *ComposeTool) Run(params map[string]any) *tools.ToolResponse {
    response1 := t.registry.ExecuteTool("tool_a", map[string]any{"input": "..."})
    response2 := t.registry.ExecuteTool("tool_b", map[string]any{"data": response1.Data})

    return tools.Success("组合执行完成", response2.Data)
}
```

---

## 📚 相关文档

- [工具响应协议](./tool-response-protocol.md) - ToolResponse 详细说明
- [文件操作工具](./file_tools.md) - 内置文件工具示例
- [Skills 知识外化](./skills-usage-guide.md) - Skills 系统集成

---

## 🤝 贡献你的工具

如果你开发了通用的工具，欢迎贡献到 HelloAgents-Go 框架：

1. Fork 项目仓库
2. 在 `hello_agents/tools/builtin/` 添加你的工具
3. 编写测试和文档
4. 提交 Pull Request
