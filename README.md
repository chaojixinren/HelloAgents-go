# HelloAgents-Go

> 🤖 生产级多智能体框架（Go 语言实现）- 工具响应协议、上下文工程、会话持久化、子代理机制等16项核心能力

[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://golang.org/dl/)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)

HelloAgents-Go 是 [HelloAgents Python 版本](https://github.com/jjyaoao/HelloAgents) 的 Go 语言忠实重实现，基于 OpenAI 原生 API 构建的生产级多智能体框架，集成了工具响应协议（ToolResponse）、上下文工程（HistoryManager/TokenCounter）、会话持久化（SessionStore）、子代理机制（TaskTool）、乐观锁（文件编辑）、熔断器（CircuitBreaker）、Skills 知识外化、TodoWrite 进度管理、DevLog 决策记录、流式输出（SSE）、异步生命周期、可观测性（TraceLogger）、日志系统（四种范式）、LLM/Agent 基类重构等 16 项核心能力，为构建复杂智能体应用提供完整的工程化支持。

## 📌 版本说明

> **重要提示**：本仓库是 HelloAgents 的 Go 语言重实现版本

- **🐍 Python 原版**：[HelloAgents](https://github.com/jjyaoao/HelloAgents)
  与 [Datawhale Hello-Agents 教程](https://github.com/datawhalechina/hello-agents) 配套的 Python 原版实现。

- **🚀 Go 版本（本仓库）**：以 Python 版本 V1.0.0 为基准，使用 Go 语言忠实重实现全部 16 项核心能力，模块与功能语义完全对齐。

- **📦 历史版本**：[Releases 页面](https://github.com/jjyaoao/HelloAgents/releases)
  提供 Python 版本从 v0.1.1 到 v0.2.9 的所有版本。

## 🚀 快速开始

### 安装

```bash
git clone https://github.com/your-repo/helloagents-go.git
cd helloagents-go
go mod download
```

### 基本使用

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

	out, err := agent.Run("分析项目结构并生成报告", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
```

### 环境配置

创建 `.env` 文件：
```bash
LLM_MODEL_ID=your-model-name
LLM_API_KEY=your-api-key-here
LLM_BASE_URL=your-api-base-url
LLM_TIMEOUT=60
```

```go
// 自动检测 provider
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
fmt.Printf("检测到的 provider: %s\n", llm.Provider)
```

> 💡 **智能检测**: 框架会根据 API 密钥格式和 Base URL 自动选择合适的 provider

### 支持的 LLM 提供商

框架基于 **3 种适配器** 支持所有主流 LLM 服务：

#### 1. OpenAI 兼容适配器（默认）

支持所有提供 OpenAI 兼容接口的服务：

| 提供商类型   | 示例服务                               | 配置示例                             |
| ------------ | -------------------------------------- | ------------------------------------ |
| **云端 API** | OpenAI、DeepSeek、Qwen、Kimi、智谱 GLM | `LLM_BASE_URL=api.deepseek.com`      |
| **本地推理** | vLLM、Ollama、SGLang                   | `LLM_BASE_URL=http://localhost:8000` |
| **其他兼容** | 任何 OpenAI 格式接口                   | `LLM_BASE_URL=your-endpoint`         |

#### 2. Anthropic 适配器

| 提供商     | 检测条件                        | 配置示例                                 |
| ---------- | ------------------------------- | ---------------------------------------- |
| **Claude** | `base_url` 包含 `anthropic.com` | `LLM_BASE_URL=https://api.anthropic.com` |

#### 3. Gemini 适配器

| 提供商            | 检测条件                                                 | 配置示例                                                 |
| ----------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| **Google Gemini** | `base_url` 包含 `googleapis.com` 或 `generativelanguage` | `LLM_BASE_URL=https://generativelanguage.googleapis.com` |

> 💡 **自动适配**：框架根据 `base_url` 自动选择适配器，无需手动指定。

## 🏗️ 项目结构

```
helloagents-go/
├── hello_agents/              # 主包
│   ├── core/                  # 核心组件
│   │   ├── llm.go             # LLM 基类与配置
│   │   ├── llm_adapters.go    # 三种适配器（OpenAI/Anthropic/Gemini）
│   │   ├── agent.go           # Agent 基类（Function Calling 架构）
│   │   ├── config.go          # 配置管理
│   │   ├── session_store.go   # 会话持久化
│   │   ├── lifecycle.go       # 异步生命周期
│   │   ├── streaming.go       # SSE 流式输出
│   │   └── message.go         # 消息定义
│   ├── agents/                # Agent 实现
│   │   ├── simple_agent.go    # SimpleAgent
│   │   ├── react_agent.go     # ReActAgent
│   │   ├── reflection_agent.go # ReflectionAgent
│   │   ├── plan_solve_agent.go # PlanAndSolveAgent
│   │   └── factory.go         # Agent 工厂
│   ├── tools/                 # 工具系统
│   │   ├── registry.go        # 工具注册表
│   │   ├── response.go        # ToolResponse 协议
│   │   ├── circuit_breaker.go # 熔断器
│   │   ├── tool_filter.go     # 工具过滤（子代理机制）
│   │   └── builtin/           # 内置工具
│   │       ├── file_tools.go  # 文件工具（乐观锁）
│   │       ├── task_tool.go   # 子代理工具
│   │       ├── todowrite_tool.go # 进度管理
│   │       ├── devlog_tool.go # 决策日志
│   │       └── skill_tool.go  # Skills 知识外化
│   ├── context/               # 上下文工程
│   │   ├── history.go         # HistoryManager
│   │   ├── token_counter.go   # TokenCounter
│   │   ├── truncator.go       # ObservationTruncator
│   │   └── builder.go         # ContextBuilder
│   ├── observability/         # 可观测性
│   │   └── trace_logger.go    # TraceLogger
│   ├── logging/               # 日志系统
│   │   └── logging.go         # AgentLogger
│   └── skills/                # Skills 系统
│       └── loader.go          # SkillLoader
├── cmd/                       # 入口命令
├── docs/                      # 文档
├── example/                   # 示例代码
├── skills/                    # 技能文件
└── tests/                     # 测试用例
```

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/) 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

**许可证要点**：
- ✅ **署名** (Attribution): 使用时需要注明原作者
- ✅ **相同方式共享** (ShareAlike): 修改后的作品需使用相同许可证
- ⚠️ **非商业性使用** (NonCommercial): 不得用于商业目的

如需商业使用，请联系项目维护者获取授权。

## 🙏 致谢

- 感谢 [HelloAgents Python 版本](https://github.com/jjyaoao/HelloAgents) 提供的原始实现
- 感谢 [Datawhale](https://github.com/datawhalechina) 提供的优秀开源教程
- 感谢 [Hello-Agents 教程](https://github.com/datawhalechina/hello-agents) 的所有贡献者
- 感谢所有为智能体技术发展做出贡献的研究者和开发者

## 📚 文档资源

详细了解 HelloAgents-Go v1.0.0 的 16 项核心能力：

### 基础设施
- **[工具响应协议](./docs/tool-response-protocol.md)** - ToolResponse 统一返回格式
- **[上下文工程](./docs/context-engineering-guide.md)** - HistoryManager/TokenCounter/Truncator

### 核心能力
- **[可观测性](./docs/observability-guide.md)** - TraceLogger 追踪系统
- **[熔断器](./docs/circuit-breaker-guide.md)** - CircuitBreaker 容错机制
- **[会话持久化](./docs/session-persistence-guide.md)** - SessionStore 会话管理

### 增强能力
- **[子代理机制](./docs/subagent-guide.md)** - TaskTool 与 ToolFilter
- **[Skills 知识外化](./docs/skills-usage-guide.md)** - 技能系统使用指南
- **[Skills 快速开始](./docs/skills-quickstart.md)** - 3 分钟上手 Skills
- **[乐观锁](./docs/file_tools.md)** - 文件编辑工具的并发控制
- **[TodoWrite 进度管理](./docs/todowrite-usage-guide.md)** - 任务进度追踪

### 辅助功能
- **[DevLog 决策日志](./docs/devlog-guide.md)** - 开发决策记录
- **[异步生命周期](./docs/async-agent-guide.md)** - 异步 Agent 实现

### 核心架构
- **[流式输出](./docs/streaming-sse-guide.md)** - SSE 流式响应
- **[Function Calling 架构](./docs/function-calling-architecture.md)** - LLM/Agent 基类重构
- **[日志系统](./docs/logging-system-guide.md)** - 四种日志范式

### 扩展能力
- **[自定义工具扩展](./docs/custom_tools_guide.md)** - 三种工具实现方式（函数式/标准类/可展开）

---

<div align="center">

**HelloAgents-Go** - 让智能体开发变得简单而强大 🚀
</div>
