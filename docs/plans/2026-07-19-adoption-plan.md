# Adoption plan — Tuile public release

## Goal

Make Tuile easy to install, import, and measure so adoption shows up in module downloads, release artifacts, and downstream `testkit` usage — not just GitHub stars.

## Scope

**In scope**

1. **Semver + Go module discoverability** — tag `v0.1.0`, visible on pkg.go.dev
2. **Install paths** — `go install`, GitHub Release binaries
3. **Release automation** — tag-triggered workflow publishing multi-platform artifacts
4. **Adoption docs** — consumer quick start, CI snippet, metrics to watch
5. **Repo discoverability** — README badges, GitHub topics

**Out of scope**

- Opt-in telemetry in `tuile serve`
- Homebrew / Docker / MCP registry publishing
- Marketing or social posts

## Implementation units

| Unit | Files | Outcome |
|------|-------|---------|
| U1 Version + CLI | `cmd/tuile/version.go`, `cmd/tuile/main.go` | `tuile version` for support |
| U2 Release CI | `.github/workflows/release.yml` | Binaries on tag push |
| U3 Adoption guide | `docs/adoption.md` | Install + testkit + metrics |
| U4 README install | `README.md` | Badges, install section, link to adoption doc |
| U5 Ship v0.1.0 | git tag + push | pkg.go.dev + release downloads |

## Metrics to track (post-ship)

| Signal | Where |
|--------|--------|
| Module versions | [pkg.go.dev/github.com/newtosh/tuile](https://pkg.go.dev/github.com/newtosh/tuile) |
| Binary downloads | GitHub Releases per asset |
| Dependents | GitHub Insights → Dependency graph |
| Imports in the wild | GitHub code search `github.com/newtosh/tuile` in `go.mod` |
| Issues/PRs | GitHub Insights |

## Verification

```bash
go install github.com/newtosh/tuile/cmd/tuile@v0.1.0
tuile version
# After tag push: GitHub Actions release job green; release assets present
```
