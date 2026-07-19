# Testing with Tuile

Tuile ships a Go **testkit** (`github.com/newtosh/tuile/testkit`) for integration and browser smoke tests against an in-process Tuile server. Use it from downstream TUI projects to catch regressions without manually running `tuile serve`.

## Quick start (consumer project)

```go
//go:build integration

package myapp_test

import (
	"testing"

	"github.com/newtosh/tuile/testkit"
)

func TestTTYSmoke(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	sess.WaitForShell(t)
	sess.Input(t, "./myapp\n")
	sess.WaitContains(t, "ready")
}
```

### Local module replace (sibling repo)

```bash
# in consumer go.mod
replace github.com/newtosh/tuile => ../tuile
```

## Running tests

```bash
# Fast unit tests (no integration tag)
go test ./...

# Integration tests (Chrome required for browser helpers)
go test -tags=integration ./test/integration/... -count=1
```

Browser tests call `t.Skip` when Chrome/chromedp is unavailable locally. **CI should install Chromium** (see Tuile’s `.github/workflows/ci.yml`).

## Pre-commit vs CI

Browser and full integration tests are **slow** and need Chrome. Do **not** run them in pre-commit hooks — use **GitHub Actions** (or your CI) instead. Keep pre-commit for lint and fast unit tests.

## GitHub Actions (copy into consumer repo)

```yaml
jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: browser-actions/setup-chrome@v1
      - run: go test -tags=integration ./... -count=1
```

Optional agent CLI tests (claude, codex, etc.) should `t.Skip` when binaries are not on `PATH` — same pattern as Tuile’s own integration suite.

## API overview

| Helper | Purpose |
|--------|---------|
| `testkit.NewServer(t)` | In-process Tuile on ephemeral port |
| `srv.NewSession(t, workspace)` | Create PTY session |
| `sess.WaitForShell(t)` | Wait for interactive shell prompt |
| `sess.EmitMarker(t, workspace, marker)` | Emit unique output without echo false-positives |
| `sess.Input(t, text)` | Write PTY input |
| `sess.WaitContains(t, marker)` | Block until screen contains text |
| `sess.PlainScreen(t, tail)` | Fetch plain screen tail |
| `sess.ViewURL()` | Browser viewer URL |
| `sess.AssertTerminalContains(t, marker)` | chromedp smoke assert |

See `testkit/doc.go` and Tuile’s `test/integration/` for full examples.
