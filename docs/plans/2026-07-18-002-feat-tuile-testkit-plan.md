---
title: Tuile Testkit - Plan
type: feat
date: 2026-07-18
topic: tuile-testkit
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: ce-brainstorm
execution: code
---

# Tuile Testkit - Plan

## Goal Capsule

- **Objective:** Give TUI and terminal-app projects a reusable Go integration-test kit for Tuile — in-process server, headless API helpers, and browser smoke helpers — plus Tuile’s own GitHub Actions CI so regressions are caught automatically.
- **Product authority:** This brainstorm.
- **Open blockers:** None.

---

## Planning Contract

### Key Technical Decisions

- **KTD1 — Package path (resolves OQ1):** `testkit/` at repo root, import `github.com/newtosh/tuile/testkit`. Public, stable surface for downstream modules; not under `internal/`.
- **KTD2 — Server fixture API:** `testkit.NewServer(t *testing.T) *Server` returns base URL, bootstrap secret, and cleanup via `t.Cleanup`. Wraps existing `httptest` + `api.NewServer` pattern from `test/integration/browser_test.go`.
- **KTD3 — Session handle:** `Server.CreateSession(workspace string) *Session` returns `ID`, `Token`, `AttachToken` (viewer), helpers for input/screen/wait. Thin HTTP client over `/v1/*` — no direct `internal/session` imports in consumer tests.
- **KTD4 — Browser context:** `testkit.Browser(t)` returns chromedp context scoped to server origins; helpers `OpenView(sess, opts)`, `WaitTerminalText`, `TerminalText`. Reuse chromedp dependency already in module.
- **KTD5 — Parallel safety:** Each `NewServer` binds `127.0.0.1:0`; sessions use `t.TempDir()` workspaces. No package-level mutable state.
- **KTD6 — CI workflow (resolves OQ2):** Single workflow `ci.yml` with jobs `unit` and `integration`. Integration job installs Chromium via `browser-actions/setup-chrome` (or `apt` fallback). Agent CLI tests (claude/codex/opencode) keep `t.Skip` when binary missing — not required for CI green on vanilla runners.
- **KTD7 — Migration strategy:** Move shared helpers from `test/integration/browser_test.go` and `e2e_test.go` into testkit first; update integration tests file-by-file; delete duplicated private helpers last.

### Assumptions

- Module path remains `github.com/newtosh/tuile` (consumers use `replace` directive or published version when available).
- chromedp remains the browser driver; no Playwright in MVP.
- `make test` continues to exclude integration-tagged packages; `make test-integration` unchanged.

### Sequencing

1. U1–U4 (testkit package core)
2. U5 (testkit self-tests where feasible)
3. U6 (migrate integration tests — proves API)
4. U7 (GHA workflow)
5. U8 (consumer docs)

### Risks

| Risk | Mitigation |
|------|------------|
| testkit imports pull heavy deps into consumers | Keep testkit as thin HTTP wrapper; only chromedp + std + existing tuile internal wiring in Server constructor |
| GHA Chrome flaky | Pin setup-chrome action; browser tests skip locally without Chrome |
| API surface churn | Dogfood in Tuile integration tests before documenting for consumers |

---

## Implementation Units

### U1. Server fixture (`testkit/server.go`)

**Files:** `testkit/server.go`, `testkit/doc.go`

- `NewServer(t *testing.T, opts ...ServerOption) *Server`
- Fields: `URL`, `Bootstrap`, `BootSecret string`
- Options: `WithAllowedOrigins`, `WithConfig` (minimal overrides only)
- `Close` via `t.Cleanup`

**Tests:** `testkit/server_test.go` — smoke: server starts, `GET /health` OK (if exists) or create session round-trip.

**Traces:** R1, R5, R6

### U2. Session and headless API helpers (`testkit/session.go`, `testkit/screen.go`)

**Files:** `testkit/session.go`, `testkit/screen.go`

- `(*Server) CreateSession(workspace string) *Session`
- `(*Session) Input(text string)`, `ScreenPlain(tail int)`, `WaitContains(marker string, timeout)`, `Resize(cols, rows)`
- `(*Server) Attach(sessionID string) string` for viewer token
- Port `waitForScreenMarker`, `postInput` patterns from `test/integration/browser_test.go`

**Tests:** `testkit/session_test.go` — in-process printf marker via wait API.

**Traces:** R2, R3, R5

### U3. Browser helpers (`testkit/browser.go`)

**Files:** `testkit/browser.go`

- `(*Session) OpenView(ctx context.Context) error` — navigate to `/view?session=&token=`
- `WaitTerminalVisible(ctx)`, `TerminalText(ctx) (string, error)`
- Skip with clear message if chromedp/Chrome unavailable (mirror `TestE2EHumanObserveTakeoverF2`)

**Tests:** covered by migrated `test/integration/browser_test.go` and `e2e_test.go`.

**Traces:** R4, SC4

### U4. Migrate integration tests to testkit

**Files:** `test/integration/*.go` (all integration test files)

- Replace `startTestServer`, `createSession`, `waitForScreenMarker`, `postInput` with testkit calls
- Keep test-specific helpers (agent spawn assertions) in integration package
- No `startTestServer` left in `test/integration` when done

**Traces:** R7, R8, SC1

### U5. GitHub Actions CI

**Files:** `.github/workflows/ci.yml`

```yaml
# shape (planning reference)
jobs:
  unit:
    runs-on: ubuntu-latest
    steps: checkout, setup-go, go test ./...
  integration:
    runs-on: ubuntu-latest
    steps: checkout, setup-go, setup-chrome, go test -tags=integration ./test/integration/...
```

- Triggers: `push` to `main`, `pull_request`
- Go version matches `go.mod` (1.26.x)

**Traces:** R9, R10, R11, SC2

### U6. Consumer documentation

**Files:** `testkit/doc.go` (package doc), `docs/testing-with-tuile.md` (new), `README.md` (link)

- Minimal consumer example test
- GHA snippet to copy
- Note: pre-commit not recommended; CI only (R13)
- `replace github.com/newtosh/tuile => ../tuile` for local sibling repos

**Traces:** R12, R13, SC3

### U7. Makefile alignment (if needed)

**Files:** `Makefile`, `README.md`

- Verify `test` target excludes integration; document CI parity commands

**Traces:** R8, F3

---

## Verification Contract

```bash
# Unit (no integration tag)
make test
go test ./testkit/... -count=1

# Integration (local; Chrome required for browser tests)
make test-integration

# Full parity with CI
go test ./... -count=1
go test -tags=integration ./test/integration/... -count=1
```

Manual: push branch and confirm GHA `unit` + `integration` jobs green.

---

## Definition of Done

- [ ] `github.com/newtosh/tuile/testkit` package exported with server, session, screen, browser helpers (R1–R6)
- [ ] `test/integration` uses testkit only — no duplicated `startTestServer` (SC1)
- [ ] `.github/workflows/ci.yml` runs unit + integration on PR (SC2)
- [ ] `docs/testing-with-tuile.md` enables downstream smoke test without reading internal sources (SC3)
- [ ] Browser tests skip gracefully without Chrome locally (SC4)
- [ ] OQ1/OQ2 resolved in KTD1/KTD6; downstream consumer adoption documented separately

### Outstanding Questions (resolved in planning)

- **OQ1.** Package path: `testkit/` → `github.com/newtosh/tuile/testkit` (KTD1).
- **OQ2.** GHA runs full integration suite; agent CLI tests skip when binaries absent (KTD6).

---

## Product Contract

### Summary

Tuile will expose a **public Go testkit package** that consuming projects import to write integration and browser smoke tests against an in-process Tuile server. Tuile dogfoods the package by migrating its existing `test/integration` suite and adding a **GitHub Actions workflow** that runs the full integration suite (including chromedp browser tests with headless Chrome). Pre-commit hooks are out of scope for Tuile tests; CI is the enforcement point. Downstream projects adopt via documentation, not in this MVP slice.

### Problem Frame

Today, Tuile’s integration patterns (start server, create session, assert screen, open `/view` in a browser) live as private helpers inside `test/integration/`. Downstream verification is often manual curl/shell steps. There is no shared library for other repos, no CI workflow in Tuile, and no standard way for downstream projects to add Tuile-backed regression coverage without copying boilerplate.

The primary users are **maintainers of Go terminal/TUI projects** who want automated smoke and regression tests without running a separate `tuile serve` process in CI.

### Key Decisions

- **Go test helper package** (session-settled: user-directed — chosen over CLI generator or templates-only). Consumers import and write standard `testing` tests.
- **CI only, not pre-commit** (session-settled: user-directed). Browser and integration tests run in GitHub Actions; local pre-commit stays fast (lint/unit).
- **In-process Tuile server** (session-settled: user-directed — chosen over subprocess `tuile serve` or external service job). Matches current httptest integration pattern; deterministic and self-contained.
- **Headless API + chromedp browser helpers** (session-settled: user-directed — chosen over API-only or Playwright). Aligns with existing Tuile integration tests.
- **Tuile-only first consumer** (session-settled: user-directed). Ship and dogfood in Tuile; external repo migration deferred.
- **Single module subpackage** (inferred — chosen over separate `tuile-testkit` repo for MVP carrying cost).

### Requirements

**Testkit package (public API)**

- R1. A public Go package (under the main `tuile` module) provides a **test server fixture** that starts Tuile in-process with ephemeral listen address, bootstrap secret, and allowed origins configured for browser tests.
- R2. The package provides **session helpers**: create session in a workspace, attach/token acquisition, close session, list sessions.
- R3. The package provides **headless assertion helpers**: send PTY input, poll/wait for screen content, read plain or compact screen tail — covering the flows already used in Tuile integration tests.
- R4. The package provides **browser helpers** built on chromedp: open `/view` with session token, wait for terminal render, assert visible text or DOM state.
- R5. Helpers are safe for parallel tests (unique ports/workspaces; no global mutable server state).
- R6. Package documentation explains minimal consumer test shape and required environment (Chrome/Chromium for browser tests).

**Tuile dogfooding**

- R7. Existing `test/integration` tests migrate to the testkit package; duplicated private helpers are removed or thin-wrapped.
- R8. Integration tests retain `//go:build integration` tag and `make test-integration` entry point.

**GitHub Actions CI**

- R9. Tuile repo gains a **GitHub Actions workflow** that runs on push/PR: unit tests (`go test ./...` excluding integration tag) and integration tests (`go test -tags=integration ./test/integration/...`).
- R10. The workflow installs/configures **headless Chrome or Chromium** for chromedp browser tests.
- R11. Workflow fails on integration test failure; documents skip behavior when optional CLIs (claude, codex, etc.) are absent.

**Consumer adoption (documentation only in MVP)**

- R12. README or dedicated doc section describes how a downstream Go project imports the testkit, writes a smoke test, and copies the GHA job pattern.
- R13. Doc includes explicit note that **pre-commit is not recommended** for browser integration tests; use CI instead.

### User Flows

**F1 — Tuile maintainer runs CI**

1. Developer opens PR.
2. GHA runs unit tests, then integration tests with headless Chrome.
3. Failure on regression in API or browser smoke path blocks merge.

**F2 — Downstream project author (post-MVP adoption)**

1. Author adds `tuile` module dependency and imports testkit in `integration_test` package.
2. Test starts in-process server, creates session for app binary/workspace, asserts screen or browser parity.
3. Author copies GHA workflow snippet; CI runs on push.

**F3 — Local developer**

1. `make test` for fast feedback.
2. `make test-integration` when validating Tuile-backed flows locally (Chrome required for browser tests).

### Success Criteria

- SC1. Tuile integration tests compile and pass using only the public testkit API (no duplicated `startTestServer` in test files).
- SC2. GHA workflow is green on main with integration + browser tests executing.
- SC3. A new consumer can write a minimal smoke test using documented helpers without reading Tuile’s internal test sources.
- SC4. Browser tests skip gracefully when Chrome is unavailable locally; CI provides Chrome explicitly.

### Scope Boundaries

**In scope**

- Public `testkit` Go package in Tuile module
- Migration of Tuile `test/integration` to testkit
- Tuile `.github/workflows` for unit + integration CI
- Consumer adoption documentation (copy-paste GHA pattern)

**Out of scope**

- Pre-commit hooks running Tuile integration tests
- Downstream repo workflow migration (follow-up)
- Playwright-based browser layer
- Separate published module or versioned `tuile-testkit` repo
- Non-Go language bindings
- `tuile test init` CLI generator
- Running full agent CLIs (claude/codex) as required CI gates — existing skip-if-missing behavior preserved

### Deferred

- Downstream first-class consumer jobs using testkit
- Pre-commit optional fast API smoke hook (if demand emerges)
- Composite GitHub Action for reusable workflow across repos
- Playwright helper layer
