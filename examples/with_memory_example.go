package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"helloagents-go/HelloAgents-go/agents"
	"helloagents-go/HelloAgents-go/core"
	"helloagents-go/HelloAgents-go/memory"
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

	// Create memory manager
	memManager := memory.NewMemoryManager()

	// Create tool registry with memory tools
	registry := tools.NewToolRegistry()
	registry.RegisterTool(builtin.NewCalculator(), false)
	registry.RegisterTool(builtin.NewTerminal(), false)
	registry.RegisterTool(builtin.NewMemoryTool(memManager), true)

	// Create agent with memory support
	agent := agents.NewSimpleAgent(
		"MemoryAgent",
		llm,
		"You are a helpful assistant with access to memory tools. "+
			"Use the memory tools to remember important information across conversations.",
		registry,
	)
	agent.EnableToolCalling(true)

	// Example 1: Storing information in memory
	fmt.Println("=== Example 1: Storing Information ===")
	response, err := agent.Run(context.Background(),
		"Please remember that I prefer learning by doing hands-on projects rather than reading documentation.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 2: Retrieving information
	fmt.Println("=== Example 2: Retrieving Information ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"What do you know about my learning preferences? Search my memory.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 3: Multiple memory types
	fmt.Println("=== Example 3: Different Memory Types ===")

	// Add working memory
	_, err = memManager.Add(context.Background(), memory.WorkingMemory,
		"Currently working on: Building a REST API in Go",
		0.8,
		map[string]interface{}{"project": "api", "language": "go"},
		[]string{"work", "current"},
	)
	if err != nil {
		log.Printf("Failed to add working memory: %v", err)
	}

	// Add episodic memory
	_, err = memManager.Add(context.Background(), memory.EpisodicMemory,
		"Yesterday had a great discussion about microservices architecture",
		0.7,
		map[string]interface{}{
			"when":  time.Now().Add(-24 * time.Hour),
			"where": "team meeting",
			"who":   []string{"team", "architect"},
		},
		[]string{"discussion", "architecture"},
	)
	if err != nil {
		log.Printf("Failed to add episodic memory: %v", err)
	}

	// Add semantic memory
	_, err = memManager.Add(context.Background(), memory.SemanticMemory,
		"Go's garbage collector is concurrent and runs alongside the application",
		0.9,
		map[string]interface{}{"category": "golang", "topic": "gc"},
		[]string{"fact", "golang", "gc"},
	)
	if err != nil {
		log.Printf("Failed to add semantic memory: %v", err)
	}

	// Query memories
	response, err = agent.Run(context.Background(),
		"Summarize all the information you have about me in different memory types.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", response)

	// Example 4: Memory statistics
	fmt.Println("=== Example 4: Memory Statistics ===")
	stats, err := memManager.GetStats(context.Background())
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Memory Statistics:\n")
		fmt.Printf("  Working: %d\n", stats[memory.WorkingMemory])
		fmt.Printf("  Episodic: %d\n", stats[memory.EpisodicMemory])
		fmt.Printf("  Semantic: %d\n", stats[memory.SemanticMemory])
		fmt.Printf("  Perceptual: %d\n\n", stats[memory.PerceptualMemory])
	}

	// Example 5: Searching memories
	fmt.Println("=== Example 5: Searching Memories ===")
	params := memory.MemorySearchParams{
		Query: "Go",
		Limit: 5,
	}
	results, err := memManager.Search(context.Background(), params)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d memories matching 'Go':\n", len(results))
		for i, mem := range results {
			fmt.Printf("%d. [%s] %s\n", i+1, mem.Type, mem.Content)
		}
		fmt.Println()
	}

	// Example 6: Memory consolidation
	fmt.Println("=== Example 6: Memory Consolidation ===")
	err = memManager.Consolidate(context.Background())
	if err != nil {
		log.Printf("Consolidation failed: %v", err)
	} else {
		fmt.Println("Memory consolidation completed successfully")
	}

	// Example 7: Using memory with other tools
	fmt.Println("\n=== Example 7: Memory + Calculator ===")
	agent.ClearHistory()
	response, err = agent.Run(context.Background(),
		"Calculate the compound interest on $1000 at 5% for 3 years, "+
			"then remember this result for future reference.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n", response)
}

// Note: This example demonstrates integration of memory systems with agents.
// Memory provides:
// - Persistence across conversations
// - Different memory types for different use cases
// - Search and retrieval capabilities
// - Automatic consolidation and pruning

// To run: go run examples/with_memory_example.go
