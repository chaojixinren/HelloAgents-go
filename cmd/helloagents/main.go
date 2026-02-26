package main

import (
	"fmt"
	"os"

	ha "helloagents-go/hello_agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/skills"
)

func main() {
	_ = core.LoadDotEnv(".env")
	cfg := core.FromEnv()

	if len(os.Args) < 2 {
		printUsage(cfg)
		return
	}

	switch os.Args[1] {
	case "doctor":
		runDoctor(cfg)
	case "version":
		fmt.Printf("HelloAgents-go %s\n", ha.Version)
	case "skills":
		listSkills(cfg)
	case "config":
		showConfig(cfg)
	case "help", "-h", "--help":
		printUsage(cfg)
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printUsage(cfg)
		os.Exit(1)
	}
}

func printUsage(cfg core.Config) {
	fmt.Printf("HelloAgents-go %s\n\n", ha.Version)
	fmt.Println("用法: helloagents <command>")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  doctor   检查环境配置")
	fmt.Println("  version  显示版本号")
	fmt.Println("  skills   列出可用技能")
	fmt.Println("  config   显示当前配置")
	fmt.Println("  help     显示此帮助信息")
}

func runDoctor(cfg core.Config) {
	fmt.Printf("HelloAgents-go %s\n", ha.Version)
	fmt.Printf("  Go 环境:    OK\n")
	fmt.Printf("  日志级别:   %s\n", cfg.LogLevel)
	fmt.Printf("  Trace:     %v\n", cfg.TraceEnabled)
	fmt.Printf("  Skills:    %v\n", cfg.SkillsEnabled)
	fmt.Printf("  Session:   %v\n", cfg.SessionEnabled)
	fmt.Printf("  Circuit:   %v\n", cfg.CircuitEnabled)

	envVars := []string{"LLM_MODEL_ID", "LLM_API_KEY", "LLM_BASE_URL"}
	allSet := true
	for _, env := range envVars {
		val := os.Getenv(env)
		if val == "" {
			fmt.Printf("  %-15s ❌ 未设置\n", env+":")
			allSet = false
		} else {
			display := val
			if env == "LLM_API_KEY" && len(val) > 8 {
				display = val[:4] + "****" + val[len(val)-4:]
			}
			fmt.Printf("  %-15s ✅ %s\n", env+":", display)
		}
	}

	if allSet {
		fmt.Println("\n✅ 环境配置就绪，可以使用 Agent")
	} else {
		fmt.Println("\n⚠️  部分环境变量未配置，请参考 .env.example")
	}
}

func listSkills(cfg core.Config) {
	loader, err := skills.NewSkillLoader(cfg.SkillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载技能失败: %v\n", err)
		os.Exit(1)
	}

	skillList := loader.ListSkills()
	if len(skillList) == 0 {
		fmt.Println("暂无可用技能")
		return
	}

	fmt.Printf("可用技能 (%d 个):\n\n", len(skillList))
	for _, name := range skillList {
		meta := loader.MetadataCache[name]
		fmt.Printf("  %-20s %s\n", name, meta["description"])
	}
}

func showConfig(cfg core.Config) {
	fmt.Printf("HelloAgents-go %s 配置:\n\n", ha.Version)
	cfgMap := cfg.ToMap()
	groups := []struct {
		title string
		keys  []string
	}{
		{"LLM", []string{"default_model", "default_provider", "temperature", "max_tokens"}},
		{"系统", []string{"debug", "log_level"}},
		{"上下文", []string{"context_window", "compression_threshold", "min_retain_rounds"}},
		{"工具", []string{"circuit_enabled", "circuit_failure_threshold"}},
		{"会话", []string{"session_enabled", "session_dir", "auto_save_enabled"}},
		{"观测", []string{"trace_enabled", "trace_dir"}},
		{"技能", []string{"skills_enabled", "skills_dir"}},
		{"流式", []string{"stream_enabled", "stream_buffer_size"}},
	}
	for _, group := range groups {
		fmt.Printf("[%s]\n", group.title)
		for _, key := range group.keys {
			fmt.Printf("  %-35s %v\n", key, cfgMap[key])
		}
		fmt.Println()
	}
}
