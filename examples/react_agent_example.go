package main

import (
	"context"
	"fmt"
	"log"

	"helloagents-go/HelloAgents-go/agents"
	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/tools"
	"helloagents-go/HelloAgents-go/tools/builtin"
)

func main() {
	// Initialize LLM client
	llm, err := core.NewHelloAgentsLLM(
		"gpt-3.5-turbo",
		"",
		"",
		"openai",
		0.7,
		nil,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to initialize LLM: %v", err)
	}

	// Create tool registry and register tools
	registry := tools.NewToolRegistry()
	registry.RegisterTool(builtin.NewCalculator(), false)
	registry.RegisterTool(builtin.NewTerminal(), false)

	// Create ReActAgent
	agent := agents.NewReActAgent(
		"ReActMathAgent",
		llm,
		"You are a helpful assistant that uses the ReAct (Reasoning + Acting) approach. "+
			"Think through problems step by step, then take actions using available tools.",
		registry,
	)
	agent.SetMaxSteps(15)

	// Example 1: Multi-step calculation
	fmt.Println("=== Example 1: Multi-step Calculation ===")
	response, err := agent.Run(context.Background(),
		"What is the result of: (15 * 8) + (23 * 12) - 45?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer: %s\n\n", response)

	// Example 2: Scientific calculation
	fmt.Println("=== Example 2: Scientific Calculation ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Calculate the area of a circle with radius 7.5, then multiply it by 3.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer: %s\n\n", response)

	// Example 3: Complex trigonometry
	fmt.Println("=== Example 3: Trigonometry ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Calculate sin(30°) + cos(60°) + tan(45°)")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer: %s\n\n", response)

	// Example 4: Problem with multiple steps
	fmt.Println("=== Example 4: Multi-step Problem ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"If I invest $1000 at 5% annual interest compounded yearly, how much will I have after 10 years?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer: %s\n\n", response)

	// Example 5: Custom ReAct prompt
	fmt.Println("=== Example 5: Custom ReAct Prompt ===")
	agent.ClearHistory()
	customPrompt := `You are an expert mathematician using the ReAct approach.

For each step:
1. Thought: Explain your reasoning
2. Action: Use calculator tool or state "finish"

Be thorough and show your work.`
	agent.SetCustomPrompt(customPrompt)

	response, err = agent.Run(context.Background(),
		"What is 12^3 + 8^2 - 144?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer: %s\n", response)
}

// Note: This example demonstrates the ReAct (Reasoning + Acting) paradigm.
// The agent follows a thought-action-observation loop:
// 1. Thought: Reason about the current state
// 2. Action: Decide what to do (use a tool or finish)
// 3. Observation: Get the result of the action
// 4. Repeat until the problem is solved

// To run: go run examples/react_agent_example.go
