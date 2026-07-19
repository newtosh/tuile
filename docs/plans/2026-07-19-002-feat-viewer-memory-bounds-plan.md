---
title: Viewer Memory Bounds - Plan
type: feat
date: 2026-07-19
topic: viewer-memory-bounds
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: ce-brainstorm
execution: code
---

# Viewer Memory Bounds - Plan

## Goal Capsule

- **Objective:** Harden the Tuile browser viewer (`/view`) so client memory stays bounded when many PTY sessions exist — no unbounded client stores, heap growth plateaus during long use, and practical dev-tool memory stays in the tens of MB range rather than climbing hundreds of MB over an afternoon.
- **Product authority:** This brainstorm.
- **Open blockers:** None.

## Planning Contract

**Product Contract preservation:** Unchanged — planning adds implementation detail only.

### Key Technical Decisions

- KTD1 — **Prune on list sync:** `syncSessions()` becomes the single choke point for client cleanup. After updating `knownSessions`, call `pruneClientSessionState(activeIds)` to drop stale `sessionCache` entries and ack-map keys not in the current list.
- KTD2 — **Extract pure helpers:** Move ack read/write/prune and cache prune into `web/session-state.js` (ES module) so logic is testable without loading the full viewer. `app.js` imports and calls from `syncSessions`, `closeSession`, and `acknowledgeSession`.
- KTD3 — **No CI heap assertions:** Verification is audit + integration tests for pruning behavior + a documented manual soak recipe (R9). Revisit automated heap thresholds only if soak finds regression-prone leaks.
- KTD4 — **Terminal reset on switch stays:** `connectWS()` already calls `term.reset()` before attach; audit confirms no duplicate xterm instances or retained scrollback across switches — fix only if audit finds a gap.
- KTD5 — **Timer audit:** `disconnectWS()` already clears WS/sync timers; extend audit to `loadingTimeoutTimer` and `pollTimer` lifecycle on session end/switch — clear or document intentional retention.

### Assumptions

- Session list polling remains at 2s (`POLL_MS`); no architectural change to sidebar transport.
- Only one viewer tab is in scope; no cross-tab `localStorage` coordination.
- `sessionCache` values are small (`{ token, cols, rows }`); unbounded key growth is the primary leak vector.

### Sequencing

1. U1 — extract + prune helpers (enables tests)
2. U2 — wire prune into `syncSessions` / `closeSession`
3. U3 — integration test for ack pruning
4. U4 — manual soak doc + optional dev-only size logging behind a query flag

---

## Implementation Units

### U1. Client session state module

**Files:** `web/session-state.js` (new), `web/app.js`

- `loadAckMap()` / `saveAckMap(map)` — centralize localStorage access for `tuile_session_ack`.
- `pruneAckMap(activeSessionIds)` — keep only keys present in `activeSessionIds` (Set or array).
- `pruneSessionCache(cache, activeSessionIds)` — delete map entries for IDs not in active set.
- `pruneClientSessionState({ cache, activeSessionIds })` — orchestrates both prunes.

**Tests:** `web/session-state.test.js` (new) via Node `--experimental-vm-modules` or minimal `node --test` script invoked from `Makefile` target `test-web` — scenarios: prune ack with 5 keys → 2 active; prune cache likewise; empty active list clears all.

**Traces:** R1, R2, R6

### U2. Wire pruning into viewer lifecycle

**Files:** `web/app.js`, `web/index.html` (add `<script type="module">` import if needed)

- Import helpers from `./session-state.js`.
- In `syncSessions`, after `syncSessions(body.sessions || [])`, derive `activeIds` from list and call `pruneClientSessionState`.
- In `closeSession`, keep explicit `sessionCache.delete(id)`; also prune ack for closed id (or rely on next sync — prefer immediate prune on close for responsiveness).
- Audit `attachToSession` / `connectWS` / `disconnectWS` — confirm single WS, `term.reset()` on switch, no listener duplication in `renderSessionList` reconcile path.

**Traces:** R1–R5, R7

### U3. Integration test — stale ack pruning

**Files:** `test/integration/viewer_client_state_test.go` (new), `testkit/browser.go` (optional helper)

- Start server + browser context.
- Navigate to `/view`, inject bootstrap secret (form or `localStorage` via `chromedp.Evaluate`).
- Create 3 sessions via HTTP API; wait for sidebar rows.
- Seed `localStorage.tuile_session_ack` with 3 IDs via `chromedp.Evaluate`.
- DELETE/close 2 sessions via API; wait for poll (≥3s).
- Evaluate ack map — assert only 1 key remains, matching surviving session ID.

**Traces:** R2, AE1

### U4. Manual soak verification doc

**Files:** `docs/viewer-memory-soak.md` (new), link from `CONTRIBUTING.md` or `docs/testing-with-tuile.md`

- Steps: create N sessions (script or repeated `tuile session start`), open `/view`, attach/switch M times, optional long output on one session.
- Chrome DevTools → Memory: take heap snapshot at T0, T+5m, T+30m; compare detached DOM nodes and JS heap size.
- Pass criteria aligned with R7–R8: plateau not linear climb; qualitative tens-of-MB bar.
- Optional: document `?debug=memory` query param if U2 adds periodic `console.debug` of `sessionCache.size` and `Object.keys(ack).length` (dev-only, no production UI).

**Traces:** R9, AE2–AE4

---

## Verification Contract

```bash
# Unit (Go)
make test

# Web helper unit tests (after U1)
make test-web   # or: node --test web/session-state.test.js

# Integration — ack pruning (U3)
go test -tags=integration ./test/integration/ -run TestViewerPrunesStaleAckState -count=1

# Full integration (no regression)
make test-integration

# Manual soak (U4)
# Follow docs/viewer-memory-soak.md with tuile serve running
```

---

## Definition of Done

- [ ] `syncSessions` prunes `sessionCache` and `tuile_session_ack` to current session list only (R1, R2, R6)
- [ ] Switching sessions does not leave multiple WebSockets open (R3)
- [ ] `term.reset()` runs on attach; no duplicate terminal instances (R4)
- [ ] `web/session-state.test.js` passes for prune helpers
- [ ] `TestViewerPrunesStaleAckState` integration test passes
- [ ] `docs/viewer-memory-soak.md` published and linked
- [ ] Manual soak run once on dev machine with qualitative plateau noted in PR test plan
- [ ] No CI heap threshold test added (per KTD3 unless leak found)

---

## Product Contract

### Summary

Audit client-side memory behavior in the Tuile viewer, fix unbounded or leaky state, and verify with a manual soak recipe. Formal CI memory regression tests are deferred unless the audit surfaces a subtle or recurring issue.

### Problem Frame

The viewer is designed for local development: one attached session at a time, with a session sidebar that polls every 2 seconds. As agent workflows spawn more concurrent PTY sessions, client-side caches and persistence can grow even when only one terminal is active. The user has not measured a leak yet but wants confidence before scaling up multi-session use.

Known client state surfaces from the current implementation (`web/app.js`):

- `sessionCache` — attach tokens keyed by session ID; pruned on explicit close, not when sessions disappear from the server list
- `tuile_session_ack` in `localStorage` — acknowledgment timestamps that grow without pruning
- Single xterm instance with `scrollback: 5000` — bounded per attach, but heavy output while attached still consumes memory
- One WebSocket for the active session; 2s list polling with incremental DOM reconcile

### Key Decisions

- **Fix-first over CI memory gates** *(session-settled: user-directed — chosen over automated CI heap assertions: user wants audit and fixes before formal validation infrastructure)* — confidence comes from code audit plus manual soak, not flaky `performance.memory` thresholds in CI.
- **Viewer-only scope** — server-side session memory, PTY buffer limits, and API pagination are out of scope unless the audit proves the browser is innocent.
- **Prune against live session list** — stale client entries are removed when a session ID is no longer returned by `/v1/sessions`, not by blind truncation that could break activity indicators for active sessions.

### Requirements

**Client lifecycle hygiene**

- R1. When `/v1/sessions` no longer includes a session ID, the viewer removes that ID from `sessionCache` and any in-memory derived state tied to it.
- R2. The `tuile_session_ack` map is pruned to entries whose session IDs still appear in the current session list (or a documented equivalent bound keyed to known sessions).
- R3. At most one WebSocket connection is open for the attached session; switching sessions closes the prior socket and clears attach-specific timers/listeners.
- R4. Attaching to a different session resets or reuses the terminal buffer without retaining scrollback or renderer state from the previous session beyond what xterm requires for the new attach.
- R5. Session-list polling and refresh handlers do not accumulate unbounded closures, DOM nodes, or duplicate listeners across create/remove cycles.

**Success bars (manual verification)**

- R6. Client state size scales with the number of sessions currently listed, not with historical sessions created over a dev day.
- R7. After a scripted soak (many listed sessions, repeated attach/switch, one long-running attached session with output), browser heap growth plateaus rather than climbing linearly over 30+ minutes.
- R8. A single viewer tab remains practical for local dev: memory in the tens of MB after soak, not hundreds of MB climbing over an afternoon without user action.

**Documentation**

- R9. A short manual verification recipe documents the soak steps and how to read Chrome DevTools Memory (or equivalent) to confirm R6–R8.

### Scope Boundaries

**In scope**

- `web/app.js` and related viewer assets (`web/state.js`, styles only if needed for lifecycle)
- Manual soak verification recipe (e.g. in `docs/` or `CONTRIBUTING.md` appendix)

**Out of scope**

- Automated CI memory regression tests (unless audit finds a regression-prone leak worth guarding)
- Server-side session limits, screen history caps, or API changes
- Multi-tab viewer coordination or shared-worker architecture
- Changing the 2s poll interval or sidebar activity heuristic (see `docs/plans/2026-07-18-001-feat-session-sidebar-activity-plan.md`)

### Acceptance Examples

- **AE1 — Stale session cleanup:** Ten sessions are created, then five are closed server-side. After the next list refresh, `sessionCache` and `tuile_session_ack` contain no entries for the five closed IDs.
- **AE2 — Switch churn:** User attaches to session A, then B, then C, repeating 20 times. Heap after minute 30 is within ~20% of heap after minute 5 (plateau, not linear growth).
- **AE3 — Many listed, one attached:** Twenty sessions appear in the sidebar; user observes one. Memory stays bounded relative to session count; no per-session WebSocket or terminal instance is created for non-attached rows.
- **AE4 — Long output:** One attached session receives continuous output for 30 minutes. Scrollback stays within configured bounds; memory does not grow without bound after initial plateau.

### Risks

| Risk | Mitigation |
|------|------------|
| Pruning ack state breaks activity dots | Prune only IDs absent from current list; re-ack on attach |
| Over-aggressive cache clear forces re-attach latency | Cache tokens only for listed sessions; re-fetch on attach if missing |
| Manual soak is environment-dependent | Document steps and qualitative plateau check, not absolute MB thresholds |

### Outstanding Questions

None — audit findings may add implementation detail during execution.
