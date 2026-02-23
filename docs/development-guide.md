# 开发指南

## 环境要求

- Go 1.21+
- 可访问的 LLM API

## 本地启动

1. 安装依赖：
```bash
go mod download
```

2. 准备 `.env`（可参考 `.env.example`）：
```bash
LLM_MODEL_ID=...
LLM_API_KEY=...
LLM_BASE_URL=...
```

3. 构建与运行：
```bash
go test ./...
go run ./cmd/helloagents
```

## 开发约定

- 保持与 Python 实现语义一致，不随意新增或删减功能
- 新增方法尽量提供 Python 同名语义别名（如 `ToDict`）
- 错误信息与状态码保持一致（`ToolErrorCode`）
- 工具输出遵循 `ToolResponse` 协议

## 常见开发流程

1. 在 Python 版本定位目标行为
2. 在 Go 对应模块实现相同行为
3. 运行：
```bash
gofmt -w .
go test ./...
go vet ./...
```
4. 补充必要文档（README + docs）

## 调试建议

- 打开 trace 观察生命周期与工具调用链路
- 优先检查：
  - 参数类型转换
  - tool_call 解析结果
  - session metadata（steps/tokens）
  - 历史与流式事件是否一致
