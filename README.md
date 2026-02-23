# HelloAgents-Go

> HelloAgents Python 版本的 Go 语言实现，保持模块与功能语义对齐。

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)

> ⚠️ 项目仍在迭代中，API 可能继续演进。

## 项目说明

HelloAgents-Go 目标是对齐 [HelloAgents](https://github.com/jjyaoao/HelloAgents) Python 版，在 Go 中提供：
- 多 Agent 范式（Simple / ReAct / Reflection / PlanSolve）
- 工具系统（注册、执行、熔断、过滤）
- 会话持久化与 Trace 观测
- Skills、Task/TodoWrite/DevLog 等能力

## 快速开始

### 1. 安装

```bash
git clone https://github.com/your-repo/helloagents-go.git
cd helloagents-go
go mod download
```

### 2. 配置 `.env`

```bash
LLM_MODEL_ID=your-model-name
LLM_API_KEY=your-api-key
LLM_BASE_URL=your-api-base-url
LLM_TIMEOUT=60
```

### 3. 最小示例

```go
package main

import (
	"fmt"
	"log"

	"helloagents-go/hello_agents/agents"
	"helloagents-go/hello_agents/core"
)

func main() {
	llm, err := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := agents.NewSimpleAgent("assistant", llm, "你是一个有用的AI助手", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	out, err := agent.Run("你好！", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
```

## 文档

- [文档总览](docs/overview.md)
- [系统架构](docs/system-architecture.md)
- [开发指南](docs/development-guide.md)
- [API 参考](docs/api-reference.md)
- [贡献指南](docs/contributing.md)
- [社区行为准则](docs/code-of-conduct.md)
- [安全策略](docs/security-policy.md)

## 许可证

[CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/)

## 致谢

- [HelloAgents Python 版本](https://github.com/jjyaoao/HelloAgents)
- [Datawhale Hello-Agents 教程](https://github.com/datawhalechina/hello-agents)
