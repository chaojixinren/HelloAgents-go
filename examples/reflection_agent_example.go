package main

import (
	"context"
	"fmt"
	"log"

	"helloagents-go/HelloAgents-go/agents"
	"helloagents-go/HelloAgents-go/core"
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

	// Create ReflectionAgent
	agent := agents.NewReflectionAgent(
		"ReflectionAgent",
		llm,
		"You are an expert writer and analyst. Your goal is to provide high-quality, well-reasoned responses.",
	)
	agent.SetMaxIterations(3)

	// Example 1: Code generation with reflection
	fmt.Println("=== Example 1: Code Generation ===")
	response, err := agent.Run(context.Background(),
		"Write a Go function that implements binary search on a sorted slice of integers.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Result:\n%s\n\n", response)

	// Example 2: Essay writing
	fmt.Println("=== Example 2: Essay Writing ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Write a short essay (200-300 words) about the importance of lifelong learning in the modern world.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Result:\n%s\n\n", response)

	// Example 3: Problem-solving
	fmt.Println("=== Example 3: Problem-Solving ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Explain how to optimize a slow database query. Provide specific techniques and examples.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Result:\n%s\n\n", response)

	// Example 4: Custom reflection prompts
	fmt.Println("=== Example 4: Custom Reflection Prompts ===")
	agent.ClearHistory()

	// Set custom prompts for each phase
	agent.SetCustomPrompt("initial",
		"Please provide a comprehensive response to the following request:\n\n%s\n\n"+
			"Take your time to think through this carefully and provide a well-structured answer.")

	agent.SetCustomPrompt("reflect",
		"Original request:\n%s\n\nCurrent response:\n%s\n\n"+
			"Please critically analyze this response. Consider:\n"+
			"1. What are the strengths of this response?\n"+
			"2. What could be improved?\n"+
			"3. Is there any missing information?\n"+
			"4. Are there any errors or inaccuracies?\n\n"+
			"Provide specific, actionable feedback.")

	agent.SetCustomPrompt("refine",
		"Original request:\n%s\n\nPrevious response:\n%s\n\nFeedback:\n%s\n\n"+
			"Please revise and improve your response based on the feedback above. "+
			"Address all the points raised and provide a better answer.")

	response, err = agent.Run(context.Background(),
		"What are the best practices for writing clean, maintainable code?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Result:\n%s\n\n", response)

	// Example 5: Adjust iteration count
	fmt.Println("=== Example 5: Adjust Iteration Count ===")
	agent.ClearHistory()
	agent.SetMaxIterations(5)

	response, err = agent.Run(context.Background(),
		"Create a detailed lesson plan for teaching recursion to beginner programmers.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Result:\n%s\n", response)

	// Show iteration info
	fmt.Printf("\nReflection iterations used: %d\n", agent.GetMaxIterations())
}

// Note: This example demonstrates the Reflection pattern for iterative improvement.
// The agent follows a reflect-improve cycle:
// 1. Generate initial answer
// 2. Reflect on strengths and weaknesses
// 3. Refine based on feedback
// 4. Repeat for specified iterations

// To run: go run examples/reflection_agent_example.go
