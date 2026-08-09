**English** | [简体中文](README_CN.md)

# HelloAgents-Go

> 🤖 Production-Grade Multi-Agent Framework (Go Implementation) - 16 core capabilities including Tool Response Protocol, Context Engineering, Session Persistence, Sub-Agent Mechanism, and more

[![CI](https://github.com/xiaoyulox/HelloAgents-go/actions/workflows/ci.yml/badge.svg)](https://github.com/xiaoyulox/HelloAgents-go/actions/workflows/ci.yml)
[![Release](https://github.com/xiaoyulox/HelloAgents-go/actions/workflows/release.yml/badge.svg)](https://github.com/xiaoyulox/HelloAgents-go/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/xiaoyulox/HelloAgents-go.svg)](https://github.com/xiaoyulox/HelloAgents-go/releases)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://golang.org/dl/)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)

HelloAgents-Go is a faithful Go reimplementation of the [HelloAgents Python version](https://github.com/jjyaoao/HelloAgents), a production-grade multi-agent framework built on the native OpenAI API. It integrates 16 core capabilities including Tool Response Protocol (ToolResponse), Context Engineering (HistoryManager/TokenCounter), Session Persistence (SessionStore), Sub-Agent Mechanism (TaskTool), Optimistic Locking (File Editing), Circuit Breaker (CircuitBreaker), Skills Externalization, TodoWrite Progress Management, DevLog Decision Recording, Streaming Output (SSE), Async Lifecycle, Observability (TraceLogger), Logging System (Four Paradigms), LLM/Agent Base Class Refactoring, providing comprehensive engineering support for building complex agent applications.

## 📌 Version Notes

> **Important Notice**: This repository is the Go reimplementation of HelloAgents

- **🐍 Python Original**: [HelloAgents](https://github.com/jjyaoao/HelloAgents)
  The original Python implementation paired with the [Datawhale Hello-Agents Tutorial](https://github.com/datawhalechina/hello-agents).

- **🚀 Go Version (This Repository)**: Based on Python version V1.0.0, faithfully reimplements all 16 core capabilities using Go, with fully aligned module and functional semantics.

- **📦 Historical Versions**: [Releases Page](https://github.com/jjyaoao/HelloAgents/releases)
  Provides all Python versions from v0.1.1 to v0.2.9.

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/your-repo/helloagents-go.git
cd helloagents-go
go mod download
```

### Basic Usage

```go
package main

import (
	"fmt"
	"log"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
	"helloagents-go/hello_agents/tools"
	"helloagents-go/hello_agents/tools/builtin"
)

func main() {
	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	registry := tools.NewToolRegistry(nil)
	registry.RegisterTool(builtin.NewReadTool("./", registry), false)
	registry.RegisterTool(builtin.NewWriteTool("./"), false)
	registry.RegisterTool(builtin.NewTodoWriteTool("./", "memory/todos"), false)

	agent, err := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)
	if err != nil {
		log.Fatal(err)
	}

	out, err := agent.Run("Analyze project structure and generate report", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
```

### Environment Configuration

Create a `.env` file:
```bash
LLM_MODEL_ID=your-model-name
LLM_API_KEY=your-api-key-here
LLM_BASE_URL=your-api-base-url
LLM_TIMEOUT=60
```

```go
// Auto-detect provider
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
fmt.Printf("Detected provider: %s\n", llm.Provider)
```

> 💡 **Smart Detection**: The framework automatically selects the appropriate provider based on the API key format and Base URL

### Supported LLM Providers

The framework supports all major LLM services through **3 adapters**:

#### 1. OpenAI-Compatible Adapter (Default)

Supports all services providing OpenAI-compatible interfaces:

| Provider Type | Example Services                      | Configuration Example                  |
| ------------- | ------------------------------------- | -------------------------------------- |
| **Cloud API** | OpenAI, DeepSeek, Qwen, Kimi, GLM     | `LLM_BASE_URL=api.deepseek.com`        |
| **Local Inference** | vLLM, Ollama, SGLang            | `LLM_BASE_URL=http://localhost:8000`   |
| **Other Compatible** | Any OpenAI-format interface     | `LLM_BASE_URL=your-endpoint`           |

#### 2. Anthropic Adapter

| Provider   | Detection Condition                   | Configuration Example                    |
| ---------- | ------------------------------------- | ---------------------------------------- |
| **Claude** | `base_url` contains `anthropic.com`   | `LLM_BASE_URL=https://api.anthropic.com` |

#### 3. Gemini Adapter

| Provider          | Detection Condition                                                  | Configuration Example                                     |
| ----------------- | -------------------------------------------------------------------- | --------------------------------------------------------- |
| **Google Gemini** | `base_url` contains `googleapis.com` or `generativelanguage`         | `LLM_BASE_URL=https://generativelanguage.googleapis.com`  |

> 💡 **Auto-Adaptation**: The framework automatically selects the adapter based on `base_url`, no manual configuration required.

## 🏗️ Project Structure

```
helloagents-go/
├── hello_agents/              # Main package
│   ├── core/                  # Core components
│   │   ├── llm.go             # LLM base class and configuration
│   │   ├── llm_adapters.go    # Three adapters (OpenAI/Anthropic/Gemini)
│   │   ├── agent.go           # Agent base class (Function Calling architecture)
│   │   ├── config.go          # Configuration management
│   │   ├── session_store.go   # Session persistence
│   │   ├── lifecycle.go       # Async lifecycle
│   │   ├── streaming.go       # SSE streaming output
│   │   └── message.go         # Message definitions
│   ├── agents/                # Agent implementations
│   │   ├── simple_agent.go    # SimpleAgent
│   │   ├── react_agent.go     # ReActAgent
│   │   ├── reflection_agent.go # ReflectionAgent
│   │   ├── plan_solve_agent.go # PlanAndSolveAgent
│   │   └── factory.go         # Agent factory
│   ├── tools/                 # Tool system
│   │   ├── registry.go        # Tool registry
│   │   ├── response.go        # ToolResponse protocol
│   │   ├── circuit_breaker.go # Circuit breaker
│   │   ├── tool_filter.go     # Tool filtering (sub-agent mechanism)
│   │   └── builtin/           # Built-in tools
│   │       ├── file_tools.go  # File tools (optimistic locking)
│   │       ├── task_tool.go   # Sub-agent tool
│   │       ├── todowrite_tool.go # Progress management
│   │       ├── devlog_tool.go # Decision logging
│   │       └── skill_tool.go  # Skills externalization
│   ├── context/               # Context engineering
│   │   ├── history.go         # HistoryManager
│   │   ├── token_counter.go   # TokenCounter
│   │   ├── truncator.go       # ObservationTruncator
│   │   └── builder.go         # ContextBuilder
│   ├── observability/         # Observability
│   │   └── trace_logger.go    # TraceLogger
│   ├── logging/               # Logging system
│   │   └── logging.go         # AgentLogger
│   └── skills/                # Skills system
│       └── loader.go          # SkillLoader
├── cmd/                       # Entry commands
├── docs/                      # Documentation
├── example/                   # Example code
├── skills/                    # Skill files
└── tests/                     # Test cases
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/) license - see the [LICENSE](LICENSE) file for details.

**License Key Points**:
- ✅ **Attribution**: You must give appropriate credit to the original author
- ✅ **ShareAlike**: Modified works must use the same license
- ⚠️ **NonCommercial**: Cannot be used for commercial purposes

For commercial use, please contact the project maintainers for authorization.

## 🙏 Acknowledgements

- Thanks to the [HelloAgents Python version](https://github.com/jjyaoao/HelloAgents) for the original implementation
- Thanks to [Datawhale](https://github.com/datawhalechina) for the excellent open-source tutorial
- Thanks to all contributors of the [Hello-Agents Tutorial](https://github.com/datawhalechina/hello-agents)
- Thanks to all researchers and developers contributing to the advancement of agent technology

## 📚 Documentation Resources

Learn more about the 16 core capabilities of HelloAgents-Go v1.0.0:

### Infrastructure
- **[Tool Response Protocol](./docs/tool-response-protocol.md)** - ToolResponse unified return format
- **[Context Engineering](./docs/context-engineering-guide.md)** - HistoryManager/TokenCounter/Truncator

### Core Capabilities
- **[Observability](./docs/observability-guide.md)** - TraceLogger tracing system
- **[Circuit Breaker](./docs/circuit-breaker-guide.md)** - CircuitBreaker fault tolerance mechanism
- **[Session Persistence](./docs/session-persistence-guide.md)** - SessionStore session management

### Enhanced Capabilities
- **[Sub-Agent Mechanism](./docs/subagent-guide.md)** - TaskTool and ToolFilter
- **[Skills Externalization](./docs/skills-usage-guide.md)** - Skills system usage guide
- **[Skills Quick Start](./docs/skills-quickstart.md)** - Get started with Skills in 3 minutes
- **[Optimistic Locking](./docs/file_tools.md)** - Concurrency control for file editing tools
- **[TodoWrite Progress Management](./docs/todowrite-usage-guide.md)** - Task progress tracking

### Auxiliary Features
- **[DevLog Decision Logging](./docs/devlog-guide.md)** - Development decision recording
- **[Async Lifecycle](./docs/async-agent-guide.md)** - Async Agent implementation

### Core Architecture
- **[Streaming Output](./docs/streaming-sse-guide.md)** - SSE streaming response
- **[Function Calling Architecture](./docs/function-calling-architecture.md)** - LLM/Agent base class refactoring
- **[Logging System](./docs/logging-system-guide.md)** - Four logging paradigms

### Extension Capabilities
- **[Custom Tool Extension](./docs/custom_tools_guide.md)** - Three tool implementation methods (functional/standard class/expandable)

---

<div align="center">

**HelloAgents-Go** - Making agent development simple and powerful 🚀
</div>
