# AGENTS.md

## Cursor Cloud specific instructions

This is a **Go library/SDK** (HelloAgents-Go) — there is no web server or database to start.

### Quick reference

| Task | Command |
|------|---------|
| Install deps | `go mod download` |
| Run all tests | `go test ./...` |
| Lint (vet) | `go vet ./...` |
| Format check | `gofmt -l .` |
| Format fix | `gofmt -w .` |
| Build CLI | `go build -o helloagents ./cmd/helloagents` |
| Run CLI | `go run ./cmd/helloagents` |
| Doctor check | `go run ./cmd/helloagents doctor` |

### Notes

- Go 1.22 is required (per `go.mod`). The VM ships with Go 1.22.2.
- All tests are pure unit tests — no external services (databases, LLM APIs) are needed to run them.
- The CLI (`cmd/helloagents/main.go`) is a minimal scaffold that prints version info and a `doctor` check. It reads `.env` for LLM config but runs fine without it.
- To use agents at runtime (not tests), you need `LLM_API_KEY`, `LLM_MODEL_ID`, and `LLM_BASE_URL` env vars — see `.env.example`.
- Development conventions are documented in `docs/development-guide.md` and `docs/contributing.md`.
