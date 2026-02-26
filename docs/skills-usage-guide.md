# Skills 知识外化系统使用指南

> 让 Agent 按需加载领域知识，无需 fine-tuning，节省 85% Token

---

## 🎯 什么是 Skills？

Skills 是 HelloAgents-Go 的知识外化系统，允许你将领域知识写成独立的 Markdown 文件，Agent 会在需要时自动加载。

**核心优势**：
- ✅ **零配置**：创建 `skills/` 目录即可自动激活
- ✅ **按需加载**：启动时只加载元数据，使用时才加载完整内容
- ✅ **Token 节省**：20 个技能场景下节省 85% Token（40K → 6K）
- ✅ **人类可编辑**：纯文本 Markdown，支持版本控制
- ✅ **团队协作**：技能文件独立管理，Git 友好

---

## 🚀 快速开始

### 1. 创建技能目录

```bash
mkdir skills
```

### 2. 创建第一个技能

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

// 创建 Agent（框架会自动检测 skills/ 目录）
llm, _ := core.NewHelloAgentsLLM("", "", "", 0.7, nil, nil, nil)
registry := tools.NewToolRegistry(nil)

agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)

// Agent 会自动加载 pdf 技能
result, _ := agent.Run("帮我提取 report.pdf 的文本内容", nil)
```

**就这么简单！** 🎉

---

## 📁 目录结构

```
your-project/
├── skills/                    # ← 技能目录（自动检测）
│   ├── pdf/
│   │   └── SKILL.md          # ← 必需文件
│   ├── code-review/
│   │   └── SKILL.md
│   └── mcp-builder/
│       └── SKILL.md
└── main.go                    # ← 你的代码
```

---

## 📝 SKILL.md 格式

```markdown
---
name: 技能名称
description: 简短描述（< 100 字符）
---

# 技能标题

详细内容...

$ARGUMENTS
```

| 字段 | 必需 | 说明 |
|-----|------|------|
| `name` | ✅ | 技能名称，用于调用 `Skill(skill="name")` |
| `description` | ✅ | 简短描述，Agent 启动时会看到（建议 < 100 字符） |
| `$ARGUMENTS` | ⚪ | 占位符，会被替换为用户传入的参数 |

---

## 🎮 使用方式

### 方式 1：零配置（推荐）

```go
// 只要 skills/ 目录存在，框架会自动激活
agent, _ := agents.NewReActAgent("assistant", llm, "", registry, nil, nil, 0, nil)
result, _ := agent.Run("帮我处理 PDF 文件", nil)
```

### 方式 2：自定义配置

```go
config := core.DefaultConfig()
config.SkillsEnabled = true            // 是否启用（默认 True）
config.SkillsDir = "my-custom-skills"  // 自定义目录
config.SkillsAutoRegister = true       // 自动注册工具

agent, _ := agents.NewReActAgent("assistant", llm, "", registry, config, nil, 0, nil)
```

### 方式 3：手动控制

```go
import "helloagents-go/hello_agents/skills"

// 手动创建 SkillLoader
loader := skills.NewSkillLoader("skills")

// 查看可用技能
fmt.Println(loader.ListSkills())         // ["pdf", "code-review", "mcp-builder"]
fmt.Println(loader.GetDescriptions())    // 格式化的技能描述

// 手动注册到 Agent
skillTool := builtin.NewSkillTool(loader)
registry.RegisterTool(skillTool, false)
```

### 方式 4：禁用 Skills

```go
config := core.DefaultConfig()
config.SkillsEnabled = false
```

---

## 📊 性能优化

### Token 节省计算

假设有 20 个技能，每个技能 2000 tokens：

| 策略 | 启动 Token | 按需加载 | 总成本 |
|-----|-----------|-----------|--------|
| **全量加载** | 20 × 2000 = 40,000 | 0 | 40,000 |
| **渐进披露** | 20 × 100 = 2,000 | 2 × 2000 = 4,000 | **6,000** |
| **节省** | | | **85%** |

---

## ✅ 最佳实践

### 1. 技能命名

- ✅ 使用小写字母和连字符：`pdf-processing`
- ❌ 避免空格和特殊字符：`PDF Processing!`

### 2. 描述编写

- ✅ 简短明确（< 100 字符）：`Process PDF files. Use when reading, creating, or merging PDFs.`
- ❌ 避免冗长描述

### 3. 版本控制

```bash
git add skills/
git commit -m "Add PDF processing skill"
```

---

## 🔗 相关文档

- [Skills 快速开始](./skills-quickstart.md) - 3 分钟上手
- [自定义工具](./custom_tools_guide.md) - 工具扩展指南

---

**最后更新**: 2026-02-21
