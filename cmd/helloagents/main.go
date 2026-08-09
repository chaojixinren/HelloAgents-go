package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ha "helloagents-go/hello_agents"
	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/skills"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	_ = core.LoadDotEnv(".env")
	cfg := core.FromEnv()

	if len(os.Args) < 2 {
		printUsage()
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
	case "run":
		runTaskCommand(cfg, os.Args[2:])
	case "chat":
		chatREPL(cfg)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("HelloAgents-go %s\n\n", ha.Version)
	fmt.Println("用法: helloagents <command> [args]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  run \"任务\"   一次性执行任务并打印结果")
	fmt.Println("  chat         进入交互式多轮对话（输入 :quit 退出）")
	fmt.Println("  doctor       检查环境配置")
	fmt.Println("  version      显示版本号")
	fmt.Println("  skills       列出可用技能")
	fmt.Println("  config       显示当前配置")
	fmt.Println("  help         显示此帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  helloagents run \"分析当前目录结构并生成报告\"")
	fmt.Println("  helloagents chat")
	fmt.Println("  helloagents doctor")
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

// ---------------------------------------------------------------------------
// Agent commands: run & chat
// ---------------------------------------------------------------------------

// buildAgent constructs a ReActAgent wired with a sensible set of built-in
// tools (file I/O, calculator, bash, http, web search). The LLM is created
// from environment variables; an error is returned when credentials are
// missing so callers can present a friendly message.
func buildAgent(cfg core.Config) (*agents.ReActAgent, error) {
	llm, err := core.NewHelloAgentsLLM("", "", "", cfg.Temperature, cfg.MaxTokens, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM 初始化失败: %w（请检查 .env 中的 LLM_MODEL_ID / LLM_API_KEY / LLM_BASE_URL）", err)
	}

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool(".", registry), false)
	registry.RegisterTool(builtin.NewWriteTool("."), false)
	registry.RegisterTool(builtin.NewEditTool("."), false)
	registry.RegisterTool(builtin.NewTodoWriteTool(".", "memory/todos"), false)
	registry.RegisterTool(builtin.NewCalculatorTool(), false)
	registry.RegisterTool(builtin.NewBashTool(), false)
	registry.RegisterTool(builtin.NewHTTPTool(), false)
	registry.RegisterTool(builtin.NewWebSearchTool(), false)

	// An empty systemPrompt lets ReActAgent fall back to its built-in
	// Thought/Finish workflow prompt.
	agent, err := agents.NewReActAgent("assistant", llm, "", &cfg, registry, 15)
	if err != nil {
		return nil, fmt.Errorf("创建 agent 失败: %w", err)
	}
	return agent, nil
}

// runTaskCommand handles `helloagents run "task description"`.
func runTaskCommand(cfg core.Config, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: helloagents run \"任务描述\"")
		fmt.Fprintln(os.Stderr, "示例: helloagents run \"分析当前目录结构并生成报告\"")
		os.Exit(2)
	}
	task := strings.TrimSpace(strings.Join(args, " "))
	if task == "" {
		fmt.Fprintln(os.Stderr, "任务描述不能为空")
		os.Exit(2)
	}

	agent, err := buildAgent(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🤖 HelloAgents-go %s\n", ha.Version)
	fmt.Printf("📋 任务: %s\n\n", task)

	out, err := agent.Run(task, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 结果:")
	fmt.Println(out)
}

// chatREPL handles `helloagents chat` — an interactive multi-turn loop.
// The agent keeps conversation history across turns. Type :quit (or :q) to
// exit.
func chatREPL(cfg core.Config) {
	agent, err := buildAgent(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("HelloAgents-go %s 交互模式\n", ha.Version)
	fmt.Println("输入消息与助手对话，输入 :quit（或 :q）退出。")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	// Allow longer single-line input (256KB) for pasting logs/code.
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	turn := 0
	for {
		fmt.Print("你> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == ":quit" || line == ":q" || line == "exit" || line == "quit" {
			break
		}

		turn++
		out, err := agent.Run(line, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "助手> ⚠️ 出错了: %v\n\n", err)
			continue
		}
		fmt.Printf("助手> %s\n\n", out)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "输入读取错误: %v\n", err)
	}
	fmt.Printf("共 %d 轮对话。再见！\n", turn)
}
