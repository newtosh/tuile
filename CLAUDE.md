# Claude Code — Tuile

This file is for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions in this repo. **Read [AGENTS.md](AGENTS.md) first** — it is the canonical agent guide (architecture, guardrails, testing, security).

## Quick context

Tuile is a **local PTY bridge for terminal UI development**: one real terminal session per workspace, exposed over HTTP and a browser viewer. Go module: `github.com/newtosh/tuile`.

```bash
make build && cp tuile.toml.example tuile.toml && ./bin/tuile serve --force
# viewer: http://127.0.0.1:7710
```

## Claude-specific reminders

- **Read AGENTS.md** before large edits; follow scope, security, and test guardrails there.
- Use **Compound Engineering (`ce-`) skills** for structured work — do not skip planning on multi-file features:
  - `ce-plan` / `ce-work` for implementation
  - `ce-debug` for failures
  - `ce-code-review` before suggesting a PR is ready
  - `ce-commit` / `ce-commit-push-pr` only when the user asks to commit or ship
- **Do not commit or push** without explicit user request.
- **Never** put secrets in commits (`tuile.toml`, tokens, bootstrap values).
- **README screenshots:** no personal names or account-specific strings.

## Verify before handoff

```bash
make test
go vet ./...
# when touching viewer, API, or testkit:
make test-integration
```

## Where to look

| Task | Start here |
|------|------------|
| HTTP API / auth | `internal/api/` |
| Session + PTY pump | `internal/session/manager.go` |
| Screen snapshots | `internal/term/` |
| Browser viewer | `web/app.js`, `web/index.html` |
| Downstream test helpers | `testkit/` |
| Integration examples | `test/integration/` |

Full map and pitfalls: [AGENTS.md](AGENTS.md).
