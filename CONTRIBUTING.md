# Contributing to HelloAgents-Go

First off, thank you for taking the time to contribute! 🎉

HelloAgents-Go is a faithful Go reimplementation of the [Python HelloAgents](https://github.com/jjyaoao/HelloAgents) framework. This guide covers everything you need to start contributing.

> 🌐 This guide is also available alongside the project. The codebase comments are in English; user-facing docs under `docs/` are in Simplified Chinese to align with the Datawhale community.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Commit Convention](#commit-convention)
- [Pull Request Process](#pull-request-process)
- [Adding New Tools](#adding-new-tools)
- [Adding New Agents](#adding-new-agents)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

By participating, you are expected to uphold our [Code of Conduct](CODE_OF_CONDUCT.md). Please be kind and respectful.

## Getting Started

### Prerequisites

- **Go 1.22+** (1.23 recommended)
- **Git**
- An LLM API key (OpenAI / DeepSeek / Anthropic / Gemini, etc.) for end-to-end testing

### Setup

```bash
# Fork and clone
git clone https://github.com/<your-username>/HelloAgents-go.git
cd HelloAgents-go

# Install dependencies
go mod download

# Verify the build
make build

# Run the test suite
make test

# Check environment
./bin/helloagents doctor
```

Copy `.env.example` to `.env` and fill in your LLM credentials for local experimentation:

```bash
cp .env.example .env
```

## Development Workflow

1. **Pick an issue** — look for issues labeled `good first issue` or `help wanted`.
2. **Create a branch** — branch off `main`:
   ```bash
   git checkout -b feature/short-description
   # or: fix/short-description, docs/short-description
   ```
3. **Make changes** following the [Code Style](#code-style) below.
4. **Run checks locally** before pushing:
   ```bash
   make check   # runs gofmt + go vet + go test
   ```
5. **Commit** using the [Commit Convention](#commit-convention).
6. **Push and open a Pull Request** against `main`.

## Code Style

This project follows standard Go conventions plus a few project-specific rules.

### Go formatting & linting

- `gofmt` is mandatory (enforced in CI).
- `golangci-lint` is configured via [`.golangci.yml`](.golangci.yml) with `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `gosimple`, `gocritic`, `misspell`, `gofmt`.
- Run locally:
  ```bash
  make fmt vet
  ```

### Project conventions

- **Package layout** mirrors the Python original (`hello_agents/core`, `agents`, `tools`, `context`, `observability`, `logging`, `skills`). When adding a module, place it under the matching package.
- **Naming parity** — public APIs keep the Python names where practical (`HelloAgentsLLM`, `ToolResponse`, `NewReActAgent`, etc.) so users can transfer knowledge between the two implementations.
- **Tool pattern** — every tool embeds `tools.BaseTool` and calls `SetRunImpl(t.Run)` in its constructor. See [calculator.go](hello_agents/tools/builtin/calculator.go) for a minimal example and [file_tools.go](hello_agents/tools/builtin/file_tools.go) for a complex one.
- **ToolResponse protocol** — never return raw strings. Use `tools.Success(...)`, `tools.Partial(...)`, or `tools.Error(text, code, details)`. Error codes are defined in [errors.go](hello_agents/tools/errors.go).
- **No panics across tool boundaries** — `BaseTool.RunWithTiming` recovers panics, but prefer returning explicit `Error` responses.
- **Comments** — code comments in English. User-facing doc strings (tool descriptions, CLI output) may be in Simplified Chinese to match existing UX.

## Commit Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/). Each commit message should be structured as:

```
<type>(<scope>): <subject>

[optional body]
```

### Allowed types

| Type     | Use for                                                |
| -------- | ----------------------------------------------------- |
| `feat`   | A new feature                                         |
| `fix`    | A bug fix                                             |
| `docs`   | Documentation only changes                            |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf`   | Performance improvement                               |
| `test`   | Adding or correcting tests                            |
| `chore`  | Build, CI, tooling, deps                              |
| `style`  | Formatting only (no code change)                      |

### Examples

```
feat(tools): add BashTool with command allowlist
fix(llm): handle empty tool_calls array in OpenAI adapter
docs(readme): add language switcher and badges
test(context): cover HistoryManager truncation edge cases
```

> 💡 Keep the subject line under 72 characters and in the imperative mood ("add", not "added").

## Pull Request Process

1. **Open a PR** against `main` using the [PR template](.github/pull_request_template.md).
2. **Fill in the template** — summary, motivation, test plan.
3. **All CI checks must pass**: `gofmt`, `go vet`, `go test -race -cover`, `go build`.
4. **Request review** from a maintainer. At least one approval is required.
5. **Address review feedback** with new commits (do not force-push during review unless asked).
6. **Squash-merge** is the default; the merge title should follow the commit convention.

### What CI runs

See [.github/workflows/ci.yml](.github/workflows/ci.yml):
- Go 1.22 + 1.23 matrix
- `gofmt -l .` check
- `go vet ./...`
- `go test -race -cover ./...`
- `go build ./cmd/helloagents`

## Adding New Tools

1. Create a file under `hello_agents/tools/builtin/`, e.g. `my_tool.go`.
2. Embed `tools.BaseTool` and implement `Run`:
   ```go
   type MyTool struct {
       tools.BaseTool
   }

   func NewMyTool() *MyTool {
       base := tools.NewBaseTool("MyTool", "简短描述", false)
       base.Parameters = map[string]tools.ToolParameter{
           "input": {Name: "input", Type: "string", Description: "...", Required: true},
       }
       t := &MyTool{BaseTool: base}
       t.BaseTool.SetRunImpl(t.Run)
       return t
   }

   func (t *MyTool) Run(parameters map[string]any) tools.ToolResponse {
       input, _ := parameters["input"].(string)
       if input == "" {
           return tools.Error("input is required", tools.ToolErrorCodeInvalidParam, nil)
       }
       return tools.Success("done", map[string]any{"output": input}, nil, nil)
   }
   ```
3. Register it where needed: `registry.RegisterTool(builtin.NewMyTool(), false)`.
4. Add tests under `tests/test_my_tool_test.go`.
5. (Optional) Add an example under `example/my_tool_demo/`.

## Adding New Agents

1. Add the agent under `hello_agents/agents/` (e.g. `my_agent.go`).
2. Embed `*core.BaseAgent` and implement `Run(inputText string, kwargs map[string]any) (string, error)`.
3. Register it in [factory.go](hello_agents/agents/factory.go) so it is discoverable via the factory.
4. Add tests under `tests/`.

## Testing

- All new features and bug fixes **must** include tests.
- Tests live in [`tests/`](tests/) (integration/feature tests) or alongside the source file as `*_test.go` (unit tests).
- Aim to maintain or improve the current coverage — CI reports coverage via `-cover`.
- Run locally:
  ```bash
  make test          # all tests
  make cover         # coverage summary
  make cover-html    # HTML coverage report
  ```
- For race detection: `go test -race ./...` (this is what CI runs).

## Documentation

- User-facing guides live in [`docs/`](docs/) in Simplified Chinese to match the tutorial series.
- Keep [README.md](README.md) (English) and [README_CN.md](README_CN.md) (Chinese) in sync — if you change one, update the other.
- When adding a new capability, add a doc under `docs/` and link it from both READMEs.
- Update [CHANGELOG.md](CHANGELOG.md) under the `## [Unreleased]` section.

## Reporting Issues

- **Bugs** — use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml).
- **Feature requests** — use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml).
- **Security vulnerabilities** — do NOT open a public issue. See [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [CC BY-NC-SA 4.0](LICENSE) license.

---

Questions? Feel free to open a [Discussion](https://github.com/xiaoyulox/HelloAgents-go/discussions) or an issue. Happy coding! 🚀
