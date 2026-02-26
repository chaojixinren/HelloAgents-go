# AGENTS.md

## Cursor Cloud specific instructions

This is a **Go library/SDK** (HelloAgents-Go) — there is no web server or database to start.

### Quick reference

| Task | Command |
|------|---------|
| Install deps | `go mod download` |
| Run all tests | `go test ./...` |
| Run tests with race detection | `go test -race ./...` |
| Coverage report | `make cover` |
| Lint (vet) | `go vet ./...` |
| Format check | `gofmt -l .` |
| Format fix | `gofmt -w .` |
| Full check (fmt + vet + test) | `make check` |
| Build CLI | `make build` |
| Run CLI | `go run ./cmd/helloagents` |
| Doctor check | `go run ./cmd/helloagents doctor` |
| List skills | `go run ./cmd/helloagents skills` |
| Show config | `go run ./cmd/helloagents config` |

### Notes

- Go 1.22 is required (per `go.mod`). The VM ships with Go 1.22.2.
- All tests are pure unit tests — no external services (databases, LLM APIs) are needed to run them. Agent tests use `core/testutil.MockLLMAdapter`.
- To use agents at runtime (not tests), you need `LLM_API_KEY`, `LLM_MODEL_ID`, and `LLM_BASE_URL` env vars — see `.env.example`.
- The `hello_agents/logging` package wraps `log/slog` — all library logging goes through it; use `logging.SetLogger()` to override.
- A `Makefile` is available with standard targets. CI runs via `.github/workflows/ci.yml`.
- Development conventions are documented in `docs/development-guide.md` and `docs/contributing.md`.
- Examples are in `examples/` (simple, react, streaming) — they require LLM API keys to run.
