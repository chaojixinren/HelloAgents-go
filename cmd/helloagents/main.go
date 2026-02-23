package main

import (
	"fmt"
	"os"

	ha "helloagents-go/hello_agents"
	"helloagents-go/hello_agents/core"
)

func main() {
	_ = core.LoadDotEnv(".env")
	cfg := core.FromEnv()

	fmt.Printf("HelloAgents-go %s\n", ha.Version)
	fmt.Printf("log_level=%s trace_enabled=%v skills_enabled=%v\n", cfg.LogLevel, cfg.TraceEnabled, cfg.SkillsEnabled)

	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		fmt.Println("doctor: scaffold is ready")
	}
}
