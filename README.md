# HelloAgents-Go

> 🤖 从零开始构建的多智能体框架 - 轻量级、原生、教学友好（Go语言版本）

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)
[![Development Status](https://img.shields.io/badge/status-in_development-yellow.svg)](https://github.com/your-repo/helloagents-go)

> ⚠️ **注意**: 本项目目前处于**积极开发阶段**，API 可能会发生变动。

HelloAgents-Go 是 [HelloAgents](https://github.com/jjyaoao/HelloAgents) Python 版本的 Go 语言实现，基于 OpenAI 原生 API 构建的多智能体框架。

## 📋 开发进度

| 模块 | 状态 | 模块 | 状态 |
|------|------|------|------|
| Core (核心) | ✅ | Tools (工具系统) | ✅ |
| SimpleAgent | ✅ | Memory (记忆系统) | 🚧 |
| ReActAgent | ✅ | Context (上下文) | 🚧 |
| ReflectionAgent | ✅ | Evaluation (评估) | 📅 |
| PlanAndSolveAgent | ✅ | Protocols (协议) | 📅 |
| FunctionCallAgent | 🚧 | RL (强化学习) | 📅 |

✅ 已完成 | 🔨 部分完成 | 🚧 开发中 | 📅 计划中

## 🚀 快速开始

### 安装

```bash
git clone https://github.com/your-repo/helloagents-go.git
cd helloagents-go
go mod download
```

### 环境配置

创建 `.env` 文件：

```bash
LLM_MODEL_ID=your-model-name
LLM_API_KEY=your-api-key
LLM_BASE_URL=your-api-base-url
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "helloagents-go/HelloAgents-go/agents"
    "helloagents-go/HelloAgents-go/core"
)

func main() {
    llm, _ := core.NewHelloAgentsLLM("", "", "", "", 0.7, nil, nil)
    agent := agents.NewSimpleAgent("AI助手", llm, "你是一个有用的AI助手", nil)
    response, _ := agent.Run(context.Background(), "你好！")
    fmt.Println(response)
}
```

## ⚙️ 支持的 LLM 提供商

| 提供商 | 环境变量 |
|--------|----------|
| OpenAI | `OPENAI_API_KEY` |
| DeepSeek | `DEEPSEEK_API_KEY` |
| 通义千问 | `DASHSCOPE_API_KEY` |
| ModelScope | `MODELSCOPE_API_KEY` |
| Kimi | `KIMI_API_KEY` |
| 智谱AI | `ZHIPU_API_KEY` |
| Ollama | `OLLAMA_HOST` |

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

## 📄 许可证

[CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/)

## 🙏 致谢

- [HelloAgents Python 版本](https://github.com/jjyaoao/HelloAgents)
- [Datawhale Hello-Agents 教程](https://github.com/datawhalechina/hello-agents)
