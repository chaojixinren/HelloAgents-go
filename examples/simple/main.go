package main

import (
	"fmt"
	"log"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func main() {
	_ = core.LoadDotEnv(".env")

	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := agents.NewSimpleAgent("assistant", llm, "你是一个有用的AI助手", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := agent.Run("请用一句话介绍 Go 语言的优势。", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
