# 系统架构

## 架构目标

HelloAgents-Go 的核心目标是与 Python 版本保持语义一致，重点包括：
- 模块边界对齐
- Agent 行为对齐
- 工具协议与生命周期对齐
- 可观测性与会话能力对齐

## 顶层模块

- `hello_agents/core`
  - LLM 统一接口、配置、消息、生命周期、流式事件、会话存储
- `hello_agents/agents`
  - `SimpleAgent` / `ReActAgent` / `ReflectionAgent` / `PlanSolveAgent`
- `hello_agents/tools`
  - 工具基类、响应协议、错误码、注册表、熔断器、过滤器
- `hello_agents/tools/builtin`
  - 内置工具（Read/Write/Edit/MultiEdit、Task、TodoWrite、DevLog、Skill、Calculator）
- `hello_agents/context`
  - 历史管理、截断器、token 计数
- `hello_agents/observability`
  - Trace 事件记录
- `hello_agents/skills`
  - Skill 元数据与按需加载

## 运行流程（简化）

1. 读取配置（含 `.env`）
2. 初始化 `HelloAgentsLLM`
3. 创建 Agent（可注入 ToolRegistry）
4. Agent 执行：
   - 组织消息
   - 调用 LLM（可含 Function Calling）
   - 执行工具
   - 写入历史与 trace
5. 可选输出流式事件、会话保存、子代理执行

## 关键设计点

- 对齐优先：优先保证与 Python 行为一致，不做额外功能扩张
- 可替换：LLM Adapter、工具注册与过滤策略可替换
- 可观测：生命周期事件 + TraceLogger 双通道观测
- 安全执行：工具熔断、防冲突写入、路径解析与备份策略
