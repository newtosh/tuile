# Adoption guide

How to install Tuile, depend on it from another Go project, and track whether people are using it.

## Install Tuile

### Go install (recommended for CLI)

```bash
go install github.com/newtosh/tuile/cmd/tuile@v0.1.2
tuile version
```

Use `@latest` after the first release, or pin a semver tag in scripts and CI.

### Prebuilt binaries

Download `tuile` (and `tuile-mcp`) for your platform from [GitHub Releases](https://github.com/newtosh/tuile/releases).

```bash
curl -sL https://github.com/newtosh/tuile/releases/download/v0.1.2/tuile-linux-amd64 -o tuile
chmod +x tuile
```

Asset names follow `tuile-<os>-<arch>` and `tuile-mcp-<os>-<arch>`.

### From source

```bash
git clone https://github.com/newtosh/tuile.git
cd tuile
make build
cp tuile.toml.example tuile.toml
./bin/tuile serve --force
```

## Adopt the testkit in your TUI project

Tuile’s main adoption path for **other repos** is the public `testkit` package — run your terminal app against an in-process Tuile server in CI.

### 1. Add the module

```bash
go get github.com/newtosh/tuile@v0.1.2
```

Sibling repo during development:

```go
replace github.com/newtosh/tuile => ../tuile
```

### 2. Write a smoke test

```go
//go:build integration

package myapp_test

import (
	"testing"

	"github.com/newtosh/tuile/testkit"
)

func TestTTYRenders(t *testing.T) {
	srv := testkit.NewServer(t)
	dir := t.TempDir()
	sess := srv.NewSession(t, dir)
	sess.WaitForShell(t)
	sess.EmitMarker(t, dir, "myapp-ready")
	sess.Input(t, "./myapp\n")
	sess.WaitContains(t, "myapp-ready")
}
```

Full API reference: [docs/testing-with-tuile.md](testing-with-tuile.md).

### 3. CI (copy-paste)

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

Browser tests need Chrome; headless-only API tests can run without it.

## Metrics (beyond stars)

| What to watch | Why it matters |
|---------------|----------------|
| **[pkg.go.dev](https://pkg.go.dev/github.com/newtosh/tuile) versions** | Shows `go get` / module proxy adoption after you tag releases |
| **GitHub Release download counts** | CLI/binary installs per OS |
| **Dependency graph** (repo Insights) | Which public repos depend on Tuile |
| **Code search** `github.com/newtosh/tuile` in `go.mod` | Downstream imports in the wild |
| **Issues & PRs from non-maintainers** | Real integrators hitting edge cases |
| **`testkit` in consumer CI** | Strongest signal — someone trusts Tuile in their pipeline |

### Practical habits

- **Tag semver releases** (`v0.1.0`, `v0.1.1`, `v0.1.2`, …) so pkg.go.dev and `go install` stay current.
- **Keep a CHANGELOG** (or GitHub release notes) so adopters know when to bump.
- **Respond to issues** filed by downstream projects — they are your adoption funnel.

### Not recommended early on

- Stars as a primary KPI
- Mandatory phone-home telemetry in `tuile serve` (erodes trust for a local PTY bridge)

## Tell users you use Tuile

If Tuile helps your project, a line in your README or a link from your test docs helps others discover the pattern:

```markdown
Terminal integration tests use [Tuile](https://github.com/newtosh/tuile) (`testkit`).
```
