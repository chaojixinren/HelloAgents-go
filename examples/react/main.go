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

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)

	cfg := core.DefaultConfig()
	agent, err := agents.NewReActAgent("react-demo", llm, "", &cfg, registry, 10)
	if err != nil {
		log.Fatal(err)
	}

	result, err := agent.Run("计算 (15 + 27) * 3 的结果", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
