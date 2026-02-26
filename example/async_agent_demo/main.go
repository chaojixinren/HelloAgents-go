// 异步 Agent 使用示例
//
// 对应 Python 版本: examples/async_agent_demo.py
// 演示如何使用 Agent 的异步生命周期功能。
// 注意: 需要配置 LLM_API_KEY 等环境变量才能运行。
package main

import (
	"fmt"
	"log"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
)

type SearchTool struct {
	tools.BaseTool
}

func NewSearchTool() *SearchTool {
	t := &SearchTool{}
	t.Name = "Search"
	t.Description = "搜索互联网信息"
	t.Parameters = map[string]tools.ToolParameter{
		"query": {Name: "query", Type: "string", Description: "搜索关键词", Required: true},
	}
	return t
}

func (t *SearchTool) GetName() string                      { return t.Name }
func (t *SearchTool) GetDescription() string               { return t.Description }
func (t *SearchTool) GetParameters() []tools.ToolParameter { return t.BaseTool.GetParameters() }
func (t *SearchTool) RunWithTiming(p map[string]any) tools.ToolResponse {
	return t.BaseTool.RunWithTiming(p)
}
func (t *SearchTool) ARun(p map[string]any) tools.ToolResponse           { return t.Run(p) }
func (t *SearchTool) ARunWithTiming(p map[string]any) tools.ToolResponse { return t.Run(p) }

func (t *SearchTool) Run(parameters map[string]any) tools.ToolResponse {
	query, _ := parameters["query"].(string)
	return tools.Success(
		fmt.Sprintf("搜索结果：关于 '%s' 的信息...", query),
		map[string]any{"query": query},
	)
}

func main() {
	_ = core.LoadDotEnv(".env")

	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(NewSearchTool(), false)

	cfg := core.DefaultConfig()
	agent, err := agents.NewReActAgent("async-demo", llm, "", &cfg, registry, 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== 异步 Agent 示例 ===")
	fmt.Println()

	// Arun with lifecycle hooks
	hooks := core.Hooks{
		OnStart: func(e core.AgentEvent) error {
			fmt.Printf("🚀 Agent 启动: %s\n", e.AgentName)
			return nil
		},
		OnFinish: func(e core.AgentEvent) error {
			fmt.Printf("✅ Agent 完成: %v\n", e.Data["result"])
			return nil
		},
		OnError: func(e core.AgentEvent) error {
			fmt.Printf("❌ Agent 错误: %v\n", e.Data["error"])
			return nil
		},
	}

	result, err := agent.Arun("搜索 Go 语言的最新动态", hooks, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n最终结果: %s\n", result)
}
