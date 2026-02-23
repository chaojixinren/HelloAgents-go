# API 参考

本文档给出当前 Go 版本的核心 API 入口（以源码为准）。

## Core

### `core.NewHelloAgentsLLM`

```go
NewHelloAgentsLLM(
  model string,
  apiKey string,
  baseURL string,
  temperature float64,
  maxTokens *int,
  timeout *int,
  kwargs map[string]any,
) (*HelloAgentsLLM, error)
```

### `core.Config`

- 使用 `core.DefaultConfig()` 获取默认配置
- 使用 `core.FromEnv()` 从环境变量加载关键配置

## Agents

### `agents.NewSimpleAgent`

```go
NewSimpleAgent(
  name string,
  llm *core.HelloAgentsLLM,
  systemPrompt string,
  config *core.Config,
  toolRegistry *tools.ToolRegistry,
) (*SimpleAgent, error)
```

### `agents.NewReActAgent`

```go
NewReActAgent(
  name string,
  llm *core.HelloAgentsLLM,
  systemPrompt string,
  config *core.Config,
  toolRegistry *tools.ToolRegistry,
  maxSteps int,
) (*ReActAgent, error)
```

### `agents.NewReflectionAgent`

```go
NewReflectionAgent(
  name string,
  llm *core.HelloAgentsLLM,
  systemPrompt string,
  config *core.Config,
  toolRegistry *tools.ToolRegistry,
) (*ReflectionAgent, error)
```

### `agents.NewPlanSolveAgent`

```go
NewPlanSolveAgent(
  name string,
  llm *core.HelloAgentsLLM,
  systemPrompt string,
  config *core.Config,
  toolRegistry *tools.ToolRegistry,
) (*PlanSolveAgent, error)
```

## Agent 运行接口

各 Agent 通常提供：
- `Run(inputText string, kwargs map[string]any) (string, error)`
- `Arun(inputText string, hooks core.Hooks, kwargs map[string]any) (string, error)`
- `ArunStream(inputText string, kwargs map[string]any, hooks ...core.Hooks) <-chan core.AgentEvent`

## Tools

### `tools.Tool` 接口

```go
type Tool interface {
  GetName() string
  GetDescription() string
  GetParameters() []ToolParameter
  Run(parameters map[string]any) ToolResponse
  RunWithTiming(parameters map[string]any) ToolResponse
  ARun(parameters map[string]any) ToolResponse
  ARunWithTiming(parameters map[string]any) ToolResponse
}
```

### `tools.ToolRegistry`

常用方法：
- `RegisterTool(tool Tool, autoExpandArgs ...bool)`
- `RegisterFunction(funcOrName any, args ...any)`  
  支持两种调用风格：
  - `RegisterFunction(handler, name?, description?)`
  - `RegisterFunction(name, description, handler)`
- `ExecuteTool(name string, inputText string) ToolResponse`
- `ListTools() []string`

### `tools.ToolResponse`

状态：
- `success`
- `partial`
- `error`

关键字段：
- `text`（给模型消费）
- `data`（结构化载荷）
- `error` / `stats` / `context`

## Builtin Tools（常用）

- `Read`, `Write`, `Edit`, `MultiEdit`
- `Task`
- `TodoWrite`
- `DevLog`
- `Skill`
- `python_calculator`

详细参数请直接查看 `hello_agents/tools/builtin/*.go` 中 `GetParameters()`。
