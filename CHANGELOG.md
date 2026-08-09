# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Community files: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, this `CHANGELOG.md`.
- GitHub Issue templates (bug report, feature request) and a Pull Request template.
- Dependabot configuration for Go module dependency updates (`.github/dependabot.yml`).
- Release workflow (`.github/workflows/release.yml`) for tagged cross-platform binaries.
- README badge bar (Go Reference, CI, Go Report Card, License, Release) with a language switcher between `README.md` (English) and `README_CN.md` (Chinese).
- Built-in tools: `BashTool` (sandboxed command execution with allowlist), `HTTPTool` (generic HTTP client), `WebSearchTool` (web search adapter).
- CLI subcommands `chat` (interactive REPL) and `run` (one-shot task execution) in `cmd/helloagents`.

## [1.0.0] - 2026-08-07

### Added — 16 core capabilities (faithful Go reimplementation of HelloAgents Python v1.0.0)

- **Tool Response Protocol** (`ToolResponse`) — standardized tool return format with `SUCCESS`/`PARTIAL`/`ERROR` states and 15 error codes.
- **Context Engineering** — `HistoryManager`, `TokenCounter` (tiktoken-backed), `ObservationTruncator`, `ContextBuilder`.
- **Session Persistence** — `SessionStore` with auto-save and resume.
- **Sub-Agent Mechanism** — `TaskTool` + `ToolFilter` for delegating subtasks.
- **Optimistic Locking** — file edit tools with sha-based conflict detection.
- **Circuit Breaker** — per-tool failure tracking with recovery timeout.
- **Skills Externalization** — file-based `SKILL.md` loader with 17 bundled skills.
- **TodoWrite** — structured task/progress management tool.
- **DevLog** — development decision logging tool.
- **Streaming Output (SSE)** — server-sent events streaming for LLM responses.
- **Async Lifecycle** — `RunAsync` with hooks (`OnStart`/`OnFinish`/`OnError`).
- **Observability** — `TraceLogger` with HTML trace export and sanitization.
- **Logging System** — `AgentLogger` with four logging paradigms.
- **LLM/Agent Base Class Refactor** — Function Calling architecture.
- **3 LLM Adapters** — OpenAI-compatible, Anthropic (Claude), Google Gemini with auto-detection.
- **4 Agent Types** — `SimpleAgent`, `ReActAgent`, `ReflectionAgent`, `PlanAndSolveAgent` with a factory.

### Infrastructure
- `cmd/helloagents` CLI with `doctor`, `version`, `skills`, `config`, `help` subcommands.
- CI workflow on GitHub Actions (Go 1.22 + 1.23 matrix, `gofmt`, `go vet`, `go test -race -cover`, `go build`).
- `Makefile` targets: `build`, `test`, `cover`, `cover-html`, `fmt`, `vet`, `check`, `clean`, `doctor`.
- `golangci-lint` configuration (`.golangci.yml`).
- `.env.example` documenting all environment variables.
- 16 example programs under `example/` and 17 test files under `tests/`.
- 16 documentation guides under `docs/`.

## Versioning

This project follows Semantic Versioning. Given a version `MAJOR.MINOR.PATCH`:

- **MAJOR** — incompatible API changes
- **MINOR** — backward-compatible new capabilities
- **PATCH** — backward-compatible bug fixes

The `Unreleased` section at the top accumulates changes that will be released
in the next version. When a release is cut, rename `[Unreleased]` to the new
version + date, and start a fresh `[Unreleased]` section.

---

For the historical Python version changelog (v0.1.1 – v0.2.9), see the
[Releases page](https://github.com/jjyaoao/HelloAgents/releases).
