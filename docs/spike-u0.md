# U0 Engine Spike Report

**Date:** 2026-07-17  
**Engine:** [gitpod-io/xterm-go](https://github.com/gitpod-io/xterm-go)  
**Verdict:** **PASS** — proceed with xterm-go as the shared emulator (KTD2). No fallback to Node `@xterm/headless` or `libghostty-vt` required.

## What was validated

| Scenario | Test | Result |
|----------|------|--------|
| Colored TUI panel (Ratatui-style borders) | `TestColoredTUIPanel` | Pass |
| Resize reflow (AE5-shaped) | `TestResizeReflow` | Pass |
| Alternate screen buffer | `TestAltScreenSwitch` | Pass |
| Malformed UTF-8 resilience | `TestMalformedUTF8DoesNotPanic` | Pass |
| Per-cell fg/bg/attrs capture | `TestSnapshotCellsIncludeColorAndAttrs` | Pass |
| Scroll region metadata | `TestSnapshotIncludesScrollRegion` | Pass |

## Grid parity with browser xterm.js

Tuile uses the same PTY byte stream for:

1. **Headless API** — `xterm-go` structured snapshots (`internal/term`)
2. **Browser viewer** — xterm.js over WebSocket + optional raw replay

Both consumers parse identical ANSI/VT sequences from one PTY. Formal golden-file comparison against `@xterm/headless` was not automated in CI; synthetic Ink/Ratatui-style sequences above exercise the same code paths used by Claude Code and Codex integration tests.

## Fallback triggers (not activated)

- **KTD2b** Node `@xterm/headless` sidecar — only if xterm-go fails Ink/Ratatui parity in production CLIs
- **libghostty-vt** CGo — last resort if both xterm paths fail

## Commands

```bash
go test ./internal/term/...
make test-integration   # claude/codex spawn when binaries on PATH
```
