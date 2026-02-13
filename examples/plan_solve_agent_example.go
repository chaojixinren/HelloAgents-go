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

	// Create PlanAndSolveAgent
	agent := agents.NewPlanAndSolveAgent(
		"PlanAndSolveAgent",
		llm,
		"You are a systematic problem solver who breaks down complex tasks into manageable steps.",
	)

	// Example 1: Research task
	fmt.Println("=== Example 1: Research Task ===")
	response, err := agent.Run(context.Background(),
		"Research and explain the key differences between SQL and NoSQL databases, with examples of when to use each.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer:\n%s\n\n", response)

	// Example 2: Tutorial creation
	fmt.Println("=== Example 2: Tutorial Creation ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Create a step-by-step tutorial for setting up a REST API using Go and the Gin framework.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer:\n%s\n\n", response)

	// Example 3: Code architecture design
	fmt.Println("=== Example 3: Architecture Design ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Design the architecture for a scalable microservices-based e-commerce platform. "+
			"Include key components, data flow, and technology recommendations.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer:\n%s\n\n", response)

	// Example 4: Learning plan
	fmt.Println("=== Example 4: Learning Plan ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Create a comprehensive 12-week learning plan for becoming a full-stack Go developer, "+
			"starting from intermediate programming knowledge.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer:\n%s\n\n", response)

	// Example 5: Custom prompts for each phase
	fmt.Println("=== Example 5: Custom Planning and Execution Prompts ===")
	agent.ClearHistory()

	// Custom planning prompt
	agent.SetCustomPrompt("plan",
		"Please create a detailed, step-by-step plan to accomplish the following task:\n\n%s\n\n"+
			"Requirements:\n"+
			"1. Break down into clear, actionable steps\n"+
			"2. Number each step (Step 1, Step 2, etc.)\n"+
			"3. Ensure logical progression\n"+
			"4. Be specific about what each step should accomplish\n"+
			"5. Estimate complexity for each step")

	// Custom execution prompt
	agent.SetCustomPrompt("execute",
		"Task: %s\n\nOverall Plan:\n%s\n\nCompleted Steps:\n%s\n\n"+
			"Current Step: %s\n\n"+
			"Please execute this step thoroughly. Provide a complete and detailed result. "+
			"Consider the overall context and how this step contributes to the final goal.")

	// Custom synthesis prompt
	agent.SetCustomPrompt("synthesize",
		"Original Task: %s\n\nPlan:\n%s\n\nAll Steps Completed:\n%s\n\n"+
			"Please synthesize all the work into a comprehensive final response. "+
			"Create a well-structured answer that directly addresses the original task, "+
			"incorporating all the insights and results from each step.")

	response, err = agent.Run(context.Background(),
		"Explain how to build a real-time chat application using WebSockets in Go.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Final Answer:\n%s\n", response)
}

// Note: This example demonstrates the Plan-and-Solve approach.
// The agent separates planning and execution:
// 1. Planning phase: Break down the problem into steps
// 2. Execution phase: Execute each step sequentially
// 3. Synthesis phase: Combine results into final answer

// Benefits:
// - Better organization for complex tasks
// - Clear progress tracking
// - Easier to debug and iterate
// - More comprehensive final results

// To run: go run examples/plan_solve_agent_example.go
