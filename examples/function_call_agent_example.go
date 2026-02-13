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

	// Create FunctionCallAgent
	agent := agents.NewFunctionCallAgent(
		"FunctionMathAgent",
		llm,
		"You are a helpful assistant with access to calculator and terminal tools. "+
			"Use the calculator for mathematical computations and terminal for system commands.",
		registry,
	)

	// Example 1: Simple calculation
	fmt.Println("=== Example 1: Simple Calculation ===")
	response, err := agent.Run(context.Background(), "What is 147 * 23?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 2: Complex multi-step calculation
	fmt.Println("=== Example 2: Multi-step Calculation ===")
	response, err = agent.Run(context.Background(),
		"Calculate the area of a circle with radius 5. Then multiply it by 3.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 3: Using trigonometric functions
	fmt.Println("=== Example 3: Trigonometry ===")
	response, err = agent.Run(context.Background(),
		"Calculate sin(30 degrees) + cos(60 degrees)")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 4: List available tools
	fmt.Println("=== Example 4: Available Tools ===")
	availableTools := agent.ListTools()
	fmt.Printf("Registered tools: %v\n", availableTools)

	// Example 5: Terminal command execution
	fmt.Println("\n=== Example 5: Terminal Command ===")
	response, err = agent.Run(context.Background(),
		"Use the terminal to check the current date and time")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 6: Sequential calculations with dependencies
	fmt.Println("=== Example 6: Sequential Calculations ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"First, calculate 15^2. Then, calculate 8^3. Finally, add these two results together.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 7: Configure agent settings
	fmt.Println("=== Example 7: Agent Configuration ===")
	agent.SetMaxToolIterations(15)
	agent.SetDefaultToolChoice("auto")
	fmt.Printf("Max tool iterations: %d\n", agent.GetMaxToolIterations())
	fmt.Printf("Default tool choice: %s\n", agent.GetDefaultToolChoice())
	fmt.Printf("Tool calling enabled: %v\n", agent.IsToolCallingEnabled())

	// Example 8: Disable tool calling temporarily
	fmt.Println("\n=== Example 8: Disable Tool Calling ===")
	agent.EnableToolCalling(false)
	response, err = agent.Run(context.Background(),
		"What is 23 * 45 without using any tools?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n", response)

	// Re-enable tool calling
	agent.EnableToolCalling(true)
}

// Note: This example demonstrates OpenAI's native function calling mechanism.
// The FunctionCallAgent automatically:
// 1. Builds JSON schemas for all registered tools
// 2. Handles multi-round tool call iterations
// 3. Converts parameter types automatically
// 4. Manages conversation history with tool results

// To run: go run examples/function_call_agent_example.go
