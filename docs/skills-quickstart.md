# Skills 快速开始

> 3 分钟上手 Skills 知识外化系统

---

## 什么是 Skills？

Skills 让 Agent 按需加载领域知识，无需修改代码，节省 85% Token。

---

## 快速开始

### 1. 创建技能目录

```bash
mkdir skills
```

### 2. 创建技能文件

创建 `skills/pdf/SKILL.md`：

```markdown
---
name: pdf
description: Process PDF files. Use when reading, creating, or merging PDFs.
---

# PDF Processing Skill

## Reading PDFs
Use pdftotext: `pdftotext input.pdf -`

## Creating PDFs
Use pandoc: `pandoc input.md -o output.pdf`

$ARGUMENTS
```

### 3. 使用 Agent

```go
import (
    "helloagents-go/hello_agents/agents"
    "helloagents-go/hello_agents/core"
    "helloagents-go/hello_agents/tools"
)

// 创建 Agent（自动检测 skills/ 目录）
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
registry := tools.NewToolRegistry(nil)

agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// Agent 会自动加载 pdf 技能
result, _ := agent.Run("帮我提取 report.pdf 的文本内容", nil)
```

**完成！** 🎉

---

## 核心优势

- ✅ **零配置**：创建 `skills/` 目录即可
- ✅ **按需加载**：节省 85% Token
- ✅ **人类可编辑**：纯文本 Markdown
- ✅ **团队协作**：Git 友好

---

## 目录结构

```
your-project/
├── skills/              # ← 创建这个目录
│   ├── pdf/
│   │   └── SKILL.md    # ← 技能定义
│   ├── code-review/
│   │   └── SKILL.md
│   └── mcp-builder/
│       └── SKILL.md
└── main.go
```

---

## SKILL.md 格式

```markdown
---
name: 技能名称
description: 简短描述（< 100 字符）
---

# 技能标题

详细内容...

$ARGUMENTS
```

**必需字段**：
- `name`：技能名称
- `description`：简短描述

---

## 配置选项

```go
import "helloagents-go/hello_agents/core"

config := core.DefaultConfig()
config.SkillsEnabled = true            // 是否启用（默认 true）
config.SkillsDir = "skills"            // 技能目录（默认 "skills"）
config.SkillsAutoRegister = true       // 自动注册（默认 true）
```

---

## 更多信息

查看完整文档：[Skills 使用指南](./skills-usage-guide.md)
