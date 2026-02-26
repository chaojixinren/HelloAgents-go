// 子代理机制使用示例
//
// 对应 Python 版本: examples/subagent_demo.py
// 演示如何使用 TaskTool 实现上下文隔离的子任务执行。
// 注意: 需要配置 LLM_API_KEY 等环境变量才能运行。
package main

import (
	"fmt"
	"log"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	_ = core.LoadDotEnv(".env")

	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	cfg := core.DefaultConfig()
	cfg.SubagentEnabled = true

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool(".", registry), false)
	registry.RegisterTool(builtin.NewWriteTool("."), false)

	agent, err := agents.NewReActAgent("main-agent", llm, "", &cfg, registry, 15)
	if err != nil {
		log.Fatal(err)
	}

	// 注册 TaskTool 用于子代理
	factory := func(agentType string) (any, error) {
		return agents.DefaultSubagentFactory(agentType, llm, registry, &cfg)
	}
	agent.RegisterTaskTool(factory)

	fmt.Println("=== 子代理机制示例 ===")
	fmt.Println()

	result, err := agent.Run("请使用子代理帮我完成：1) 读取 README.md 的内容摘要 2) 总结项目的主要功能", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
