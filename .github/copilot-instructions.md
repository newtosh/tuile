# Tuile — Copilot code review instructions

Tuile is a local PTY bridge for AI coding agents: Go HTTP/WebSocket server (`cmd/tuile`), embedded browser viewer (`web/`), and a public `testkit` for integration tests.

## Workflow expectations

- Routine changes use feature branches and pull requests; do not push directly to `main`.
- Keep diffs focused. Match existing naming, structure, and error-handling style in touched files.
- Web assets under `web/` are embedded at build time (`web/embed.go`); viewer/CSS/JS changes require rebuilding `tuile` to take effect.

## What to verify in reviews

- **Security:** Session tokens and bootstrap secrets must not leak in logs, errors, or client storage beyond existing patterns. Workspace paths must go through `internal/workpath` resolution (path-injection aware).
- **PTY/session lifecycle:** Closing, resize, takeover, and WebSocket attach/release paths must not leak goroutines, file descriptors, or leave sessions in inconsistent states.
- **Viewer client state:** `syncSessions()` should remain the choke point for pruning `sessionCache` and `tuile_session_ack`. Sidebar session list must stay scrollable (`.session-list` uses `flex: 1`, `min-height: 0`, `overflow-y: auto`).
- **Tests:** New behavior should have tests when practical. Prefer extending existing packages over new abstractions.

## Commands reviewers can rely on

```bash
make test          # Go unit tests
make test-web      # Node tests for web/session-state.js and style guards
make vet           # go vet
go test -tags=integration ./test/integration/...   # needs Chrome
```

Integration and browser tests may `t.Skip` when Chrome is unavailable; CI installs Chromium.

## Review tone

- Flag real bugs, security issues, race conditions, and missing tests for non-trivial logic.
- Skip nitpicks that do not change behavior (formatting-only, subjective naming) unless inconsistent with surrounding code.
- Copilot reviews are advisory; they do not block merge.
