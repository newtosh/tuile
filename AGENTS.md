# Agent guide — Tuile

Instructions for AI coding agents working in this repository.

## What this project is

**Tuile** is a local PTY bridge for terminal UI development. One workspace-bound shell or agent CLI session is shared between:

- a **headless HTTP API** (`/v1/sessions/...`) for automation, and
- a **browser viewer** (`/view`) for humans.

Module path: `github.com/newtosh/tuile`

## Repository map

| Path | Role |
|------|------|
| `cmd/tuile` | Main server CLI (`tuile serve`, `tuile session start`) |
| `cmd/tuile-mcp` | MCP server wrapping the HTTP client |
| `internal/session` | Session lifecycle, PTY pump, screen versioning |
| `internal/term` | xterm-go emulator wrapper (headless screen state) |
| `internal/api` | HTTP routes, WebSocket, auth, static viewer |
| `internal/pty` | POSIX PTY allocation and I/O |
| `internal/cli` | Agent CLI spawn drivers (`claude`, `codex`, `cursor-cli`, …) |
| `testkit/` | **Public** test helpers for downstream projects |
| `test/integration/` | Integration + browser tests (`//go:build integration`) |
| `web/` | Embedded viewer (xterm.js, app.js, styles) |
| `docs/` | User docs, testing guide, design plans |

## Build and verify

```bash
make build
make test                 # unit tests — run before finishing
make vet                  # or rely on CI
make test-integration     # needs Chrome; matches CI integration job
```

CI (`.github/workflows/ci.yml`) runs: `go mod tidy` check, `go vet`, both binary builds, unit tests, and integration tests.

## Guardrails

### Scope and style

- **Minimize diff size.** Fix only what the task requires. Do not refactor unrelated code.
- **Match existing patterns** in the package you touch (naming, error handling, test style).
- **Prefer extending** `internal/session`, `internal/api`, and `testkit` over new abstractions.
- **Do not add** dependencies without clear justification.
- **Comments** only for non-obvious behavior (concurrency, security, PTY semantics).

### Security

- Tuile bridges a **real local shell**. Treat session tokens and bootstrap secrets like passwords.
- Default bind is **loopback** (`127.0.0.1:7710`). Do not weaken origin checks or auth without explicit discussion.
- Never commit `tuile.toml`, tokens, or secrets. `tuile.toml.example` is the template.
- Do not log bearer tokens or bootstrap secrets.
- Report security issues via [SECURITY.md](SECURITY.md) (private vulnerability reporting), not public issues.

### API stability

- `/v1/` paths are the public HTTP contract. Breaking changes need deliberate versioning or migration notes.
- `testkit/` is a **public** surface for downstream repos — treat exported helpers as stable API.

### Tests

- Add or update tests for behavior changes in `internal/*` and `testkit/`.
- Integration tests use the `integration` build tag; browser tests need Chrome/chromedp.
- Agent CLI tests (`claude`, `codex`, `opencode`) **skip** when binaries are absent — do not make them required in CI.
- After session lifecycle changes, run `go test ./internal/session/...` (and consider `-count=10` for races).

### Docs and screenshots

- Update `README.md` when setup or user-visible behavior changes.
- README screenshots: **no personal names, emails, or account-specific strings.** Use login screens, in-progress agent output, or generic shell sessions.
- Put images in `docs/images/`.

### Git

- **Do not commit** unless the user explicitly asks.
- **Do not push** or open PRs unless asked.
- Follow [CONTRIBUTING.md](CONTRIBUTING.md).

## Recommended workflow (Compound Engineering skills)

This project uses the **ce-** skill suite for structured agent work. Prefer these over ad-hoc multi-step improvisation:

| Situation | Skill |
|-----------|--------|
| Vague feature or “what should we build?” | `ce-brainstorm` |
| Multi-step feature or refactor | `ce-plan` → `ce-work` |
| Implement from an approved plan | `ce-work` |
| Bug, flake, or failing test | `ce-debug` |
| Pre-PR self-review | `ce-code-review` |
| User asks to commit | `ce-commit` |
| User asks to ship / open PR | `ce-commit-push-pr` |
| Browser/viewer regression | `ce-test-browser` |
| Simplify recent changes | `ce-simplify-code` |
| Capture a durable learning | `ce-compound` |

Typical flow for non-trivial work:

1. `ce-brainstorm` or `ce-plan` for scope
2. `ce-work` to implement
3. `make test` (+ `make test-integration` when touching viewer/API paths)
4. `ce-code-review` before PR
5. `ce-commit-push-pr` when the user wants it merged

## Common pitfalls

- **Session close race:** close PTY before disposing the emulator; `pumpPTYToEmulator` runs in a goroutine.
- **Shell prompt detection:** POSIX shells vary (`$`, `#`, `%`, `❯`); see `testkit` `WaitForShell`.
- **Browser vs headless:** viewer uses xterm.js; API uses xterm-go — same PTY bytes, different renderers.
- **Downstream module path:** `github.com/newtosh/tuile` (not `tuile-dev`).

## Further reading

- [README.md](README.md) — usage and API overview
- [CONTRIBUTING.md](CONTRIBUTING.md) — human contributor guide
- [docs/testing-with-tuile.md](docs/testing-with-tuile.md) — `testkit` for consumer projects
- [NOTICES.md](NOTICES.md) — third-party licenses
