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
	fmt.Println("========================================")
	fmt.Println("  HelloAgents-Go Framework Demo")
	fmt.Println("========================================\n")

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

	fmt.Printf("Registered %d tools: %v\n\n", registry.Count(), registry.ListTools())

	// Demo 1: SimpleAgent
	fmt.Println("========================================")
	fmt.Println("Demo 1: SimpleAgent")
	fmt.Println("========================================")
	simpleAgent := agents.NewSimpleAgent(
		"SimpleAgent",
		llm,
		"You are a helpful assistant with access to calculator and terminal tools.",
		registry,
	)
	simpleAgent.EnableToolCalling(true)

	response, err := simpleAgent.Run(context.Background(),
		"What is 25 * 37 + 143?")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Demo 2: FunctionCallAgent
	fmt.Println("========================================")
	fmt.Println("Demo 2: FunctionCallAgent")
	fmt.Println("========================================")
	funcAgent := agents.NewFunctionCallAgent(
		"FunctionCallAgent",
		llm,
		"You are a helpful assistant using OpenAI function calling.",
		registry,
	)

	response, err = funcAgent.Run(context.Background(),
		"Calculate the square root of 625 and multiply by 3.")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Demo 3: ReActAgent
	fmt.Println("========================================")
	fmt.Println("Demo 3: ReActAgent")
	fmt.Println("========================================")
	reactAgent := agents.NewReActAgent(
		"ReActAgent",
		llm,
		"You are a systematic problem solver using the ReAct approach.",
		registry,
	)
	reactAgent.SetMaxSteps(10)

	response, err = reactAgent.Run(context.Background(),
		"Calculate (12^2 + 5^3) / 7")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Demo 4: ReflectionAgent
	fmt.Println("========================================")
	fmt.Println("Demo 4: ReflectionAgent")
	fmt.Println("========================================")
	reflectionAgent := agents.NewReflectionAgent(
		"ReflectionAgent",
		llm,
		"You are an expert analyst providing well-reasoned responses.",
	)
	reflectionAgent.SetMaxIterations(2)

	response, err = reflectionAgent.Run(context.Background(),
		"What are the key benefits of using Go for backend development?")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Demo 5: PlanAndSolveAgent
	fmt.Println("========================================")
	fmt.Println("Demo 5: PlanAndSolveAgent")
	fmt.Println("========================================")
	planAgent := agents.NewPlanAndSolveAgent(
		"PlanAndSolveAgent",
		llm,
		"You are a systematic planner who breaks down complex tasks.",
	)

	response, err = planAgent.Run(context.Background(),
		"Explain how to set up a CI/CD pipeline for a Go project.")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Demo 6: ToolAwareSimpleAgent
	fmt.Println("========================================")
	fmt.Println("Demo 6: ToolAwareSimpleAgent")
	fmt.Println("========================================")
	toolAwareAgent := agents.NewToolAwareSimpleAgent(
		"ToolAwareAgent",
		llm,
		"You are a helpful assistant with detailed tool call tracking.",
		registry,
	)
	toolAwareAgent.EnableToolCalling(true)
	toolAwareAgent.EnableDetailedLogging(false)

	response, err = toolAwareAgent.Run(context.Background(),
		"Use the calculator to compute 15% of 300.")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n", response)
	}

	// Show tool call statistics
	fmt.Printf("\nTool Call Summary: %v\n", toolAwareAgent.GetToolCallSummary())
	fmt.Printf("Total Tool Calls: %d\n", toolAwareAgent.GetToolCallCount())

	// Summary
	fmt.Println("\n========================================")
	fmt.Println("Demo Complete!")
	fmt.Println("========================================")
	fmt.Println("\nAgent Types Demonstrated:")
	fmt.Println("1. SimpleAgent - Basic conversation with tools")
	fmt.Println("2. FunctionCallAgent - OpenAI native function calling")
	fmt.Println("3. ReActAgent - Thought-action-observation loop")
	fmt.Println("4. ReflectionAgent - Iterative improvement")
	fmt.Println("5. PlanAndSolveAgent - Plan-then-execute approach")
	fmt.Println("6. ToolAwareSimpleAgent - Enhanced tool tracking")
	fmt.Println("\nTools Demonstrated:")
	fmt.Println("- Calculator: Mathematical expressions")
	fmt.Println("- Terminal: Safe shell command execution")
}

// Note: This is a comprehensive demonstration of the HelloAgents-Go framework.
// It showcases all major agent types and their capabilities.

// To run: go run examples/complete_demo.go

// For more focused examples, see:
// - simple_agent_example.go
// - function_call_agent_example.go
// - react_agent_example.go
// - reflection_agent_example.go
// - plan_solve_agent_example.go
// - with_memory_example.go
