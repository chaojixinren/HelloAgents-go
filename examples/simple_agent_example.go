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
	if err := registry.RegisterTool(builtin.NewCalculator(), true); err != nil {
		log.Fatalf("Failed to register calculator: %v", err)
	}
	if err := registry.RegisterTool(builtin.NewTerminal(), true); err != nil {
		log.Fatalf("Failed to register terminal: %v", err)
	}

	// Create SimpleAgent
	agent := agents.NewSimpleAgent(
		"MathAgent",
		llm,
		"You are a helpful math assistant. Use the calculator tool when you need to perform calculations.",
		registry,
	)
	agent.EnableToolCalling(true)

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	response, err := agent.Run(context.Background(), "Hello! What can you help me with?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 2: Using calculator tool
	fmt.Println("=== Example 2: Calculator Tool ===")
	response, err = agent.Run(context.Background(), "What is 25 * 37 + 143?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	response, err = agent.Run(context.Background(), "Calculate the square root of 625 and multiply it by 3.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 4: List available tools
	fmt.Println("=== Example 4: Available Tools ===")
	tools := agent.ListTools()
	fmt.Printf("Registered tools: %v\n", tools)

	// Example 5: Terminal command
	fmt.Println("\n=== Example 5: Terminal Tool ===")
	response, err = agent.Run(context.Background(), "Use the terminal to list the current directory.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 6: Clear history and start fresh
	fmt.Println("=== Example 6: Clear History ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(), "What is 15% of 200?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n", response)
}

// Note: This example requires:
// 1. A valid API key in .env file or OPENAI_API_KEY environment variable
// 2. The go-openai dependency installed
// 3. The core module to be compiled and available
//
// To run: go run examples/simple_agent_example.go
