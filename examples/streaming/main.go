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

	agent, err := agents.NewSimpleAgent("stream-demo", llm, "你是一个有用的AI助手", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	chunkCh, errCh := agent.StreamRun("用 3 个要点总结 Go 语言的特点。", nil)
	for chunk := range chunkCh {
		fmt.Print(chunk)
	}
	fmt.Println()

	for err := range errCh {
		if err != nil {
			log.Fatal(err)
		}
	}
}
