---
title: Session Sidebar Activity - Plan
type: feat
date: 2026-07-18
topic: session-sidebar-activity
artifact_contract: ce-unified-plan/v1
artifact_readiness: implementation-ready
product_contract_source: ce-brainstorm
execution: code
---

# Session Sidebar Activity - Plan

## Goal Capsule

- **Objective:** Make Tuile's browser session sidebar calm and scannable when multiple agent sessions are running — stable row order, explicit sort control, and clear activity status without DOM flicker or list jumping.
- **Product authority:** This brainstorm.
- **Open blockers:** None.

---

## Planning Contract

### Key Technical Decisions

- **KTD1 — Meaningful activity heuristic (resolves OQ1):** After each PTY write in `pumpPTYToEmulator`, compare a fingerprint of the last 5 non-empty tail lines (ANSI-stripped, whitespace-normalized) to the previous fingerprint. Bump `last_meaningful_activity_at` only when the fingerprint changes. Rationale: cheap, server-authoritative, avoids client polling every session's screen; cursor-only ANSI sequences rarely change tail text.
- **KTD2 — Deterministic server list order (R3):** `Manager.List()` collects sessions, sorts by `created_at` descending before return. Client applies its own sort from dropdown but never depends on map iteration order.
- **KTD3 — Incremental DOM reconcile (R2):** Replace `replaceChildren` full rebuild with keyed row update: match existing `li[data-session-id]`, patch text/classes/dot only; insert/remove rows when session set changes. Reorder DOM nodes only when sort comparator order differs.
- **KTD4 — Acknowledgment state (R14):** Per-browser `Map<sessionId, acknowledgedAt>` in memory, persisted to `localStorage` key `tuile_session_ack` as JSON. Acknowledge on row click sets `acknowledgedAt = now`. Recent activity when `last_meaningful_activity_at > acknowledgedAt`.
- **KTD5 — Inactive threshold (resolves OQ2):** Default **15 minutes**. Stored in `localStorage` as `tuile_session_inactive_mins` (integer). Small number input in session panel footer. Inactive when `now - last_meaningful_activity_at > threshold` (and not in recent-activity state).
- **KTD6 — Sort persistence (R9):** `localStorage` key `tuile_session_sort` = `{ field, direction }`. Fields: `created`, `label`, `id`, `duration`. Default `{ field: "created", direction: "desc" }`.
- **KTD7 — Activity dot on active session (R11 carry-forward):** No special case; user may acknowledge by re-clicking or we auto-ack on attach (same as row click). Attach already calls acknowledge.

### Assumptions

- Session list polling stays at 2s (`POLL_MS`); no new websocket for sidebar.
- `created_at` set at `Manager.Create` time; never changes.
- Duration sort uses `now - created_at` at render time (client clock).

### Sequencing

1. U1 + U2 (server metadata + API) — unblocks client activity dots
2. U3 (incremental render) — fixes flicker immediately even before dots
3. U4 + U5 + U6 (sort, dots, threshold UI)
4. U7 (tests)

### Risks

| Risk | Mitigation |
|------|------------|
| Fingerprint false negatives on same-text refresh | Acceptable; full-screen TUI redraws usually change tail |
| High poll + many sessions | Reconcile path avoids layout thrash; no extra API calls |
| Clock skew inactive state | Use relative durations from server timestamps only |

---

## Implementation Units

### U1. Server session timestamps and meaningful-activity tracking

**Files:** `internal/session/manager.go`, `internal/session/activity.go` (new), `internal/session/output.go`

- Add `CreatedAt`, `LastMeaningfulActivityAt` to `Session`; set on create; initialize both to `time.Now()`.
- In `pumpPTYToEmulator`, after `Emulator.Write`, call `sess.notePTYOutput()` which updates fingerprint + `LastMeaningfulActivityAt` when tail fingerprint changes.
- Export tail fingerprint helper using `term.TailLines` + strip ANSI (reuse or mirror `term.JoinTailText` patterns).

**Tests:** `internal/session/activity_test.go` — fingerprint changes on new line; unchanged on identical rewrite.

### U2. Enriched `GET /v1/sessions` response

**Files:** `internal/session/manager.go` (`SessionInfo`), `internal/api/router.go`, `internal/api/router_test.go`

- Extend `SessionInfo` JSON: `created_at` (RFC3339), `last_meaningful_activity_at` (RFC3339).
- `List()` returns sessions sorted `created_at` desc by default.
- Update `TestDeleteAndPruneSessions` / add test asserting field presence and stable sort.

**Traces:** R4, R5, R6, R3

### U3. Incremental session list rendering

**Files:** `web/app.js`

- Refactor `renderSessionList` → `reconcileSessionList(sortedSessions)`:
  - Build desired order from sort comparator (U4 fields available; default created desc).
  - For each id: find or create `li.session-row[data-session-id]`.
  - Update `.workspace`, `.meta`, `.session-status` classes only when data changed.
  - Move nodes to match order without recreating listeners (delegate or bind once per row).
- Keep early-exit when ids AND order AND row payload unchanged (extend current check).

**Traces:** R1, R2

### U4. Sort dropdown

**Files:** `web/index.html`, `web/app.js`, `web/style.css`

- Add `<select id="session-sort">` in panel header with options: Newest, Oldest, Agent A–Z, Agent Z–A, Session ID, Longest running, Shortest running.
- `sortSessions(list, sortKey)` pure function; read/write `tuile_session_sort` from localStorage.
- Label helper: `displayLabel(sess)` = `sess.cli || basename(workspace)`.

**Traces:** R7, R8, R9, F4

### U5. Activity status dot

**Files:** `web/app.js`, `web/style.css`

- Add `<span class="session-status" aria-label="...">` left of workspace text.
- States: `active` (recent), `idle`, `inactive` — CSS variables for colors (accent / neutral / muted).
- `computeSessionStatus(sess)` uses server timestamps + ack map + inactive threshold.
- On row click (before attach): `acknowledgeSession(id)`.

**Traces:** R10–R14, F1, F2, F3

### U6. Inactive threshold control

**Files:** `web/index.html`, `web/app.js`, `web/style.css`

- Compact input in session panel: "Inactive after [N] min" with min=1 max=1440, default 15.
- Persist `tuile_session_inactive_mins`; re-render list on change.

**Traces:** R15

### U7. Integration and regression tests

**Files:** `internal/api/router_test.go`, optional `test/integration/session_list_test.go`

- API test: create two sessions, bump activity on one, list returns distinct `last_meaningful_activity_at`.
- Manual checklist: F1–F4 with multiple shell/agent sessions via Tuile viewer.

**Traces:** SC1–SC4, F1–F4

---

## Verification Contract

```bash
# Unit
go test ./internal/session/... ./internal/api/... -count=1

# Integration (optional browser)
go test -tags=integration ./test/integration/... -count=1   # no regression

# Manual (Tuile running)
# 1. Create 3 sessions, verify stable order over 30s polling
# 2. Background output on non-focused session → dot active, order unchanged
# 3. Click row → dot idle
# 4. Change sort dropdown → order updates once, persists on reload
```

---

## Definition of Done

- [ ] `GET /v1/sessions` includes `created_at` and `last_meaningful_activity_at`; order deterministic
- [ ] Sidebar does not flicker when polling with unchanged session set (SC1)
- [ ] Activity dot reflects meaningful output; acknowledge on click (F2)
- [ ] Sort dropdown works; persists in localStorage (F4, SC3)
- [ ] Inactive threshold configurable; default 15m (R15)
- [ ] Unit tests for activity fingerprint and API fields pass
- [ ] OQ3 remains deferred; no scope creep into terminal renderer

### Outstanding Questions (deferred)

- **OQ3.** “Unseen only” activity dot toggle — revisit if active-session dots are noisy in dogfooding.

---

## Product Contract

### Summary

The session panel in Tuile's browser viewer (`/view`) will stop reordering rows on background PTY activity. Users get a sort dropdown (default: newest session first), a left-side status dot per row, and three visual states: **recent activity** (unacknowledged meaningful output), **idle** (acknowledged or no recent activity), and **inactive** (no meaningful activity for longer than a user-configurable threshold). Opening a session row acknowledges its activity.

### Problem Frame

With several shell or agent sessions open, the sidebar currently polls every two seconds and rebuilds the list when row order changes. The server returns sessions from a map with non-deterministic iteration order, so rows jump even when nothing meaningful changed. Full `replaceChildren` redraws amplify flicker. Users cannot tell which session had recent output without clicking each one, and the active/highlight state competes with reordering.

The primary user is a human operator (or developer) tailing multiple agent sessions in observe mode who needs to notice activity elsewhere without losing their place in the list.

### Key Decisions

- **Stable list order** (session-settled: user-directed — chosen over auto-bumping recently active sessions to the top). Activity is communicated only via the status dot and row styling, not reordering.
- **Meaningful PTY activity only** (session-settled: user-directed — chosen over “any PTY byte” or “unseen only”). Pure cursor/escape noise must not flash the dot.
- **Default sort: newest first by session creation time** (session-settled: user-directed).
- **Sort UI: dropdown in the session panel header** (session-settled: user-directed — chosen over multi-column headers in the narrow sidebar).
- **Acknowledge on row click** (session-settled: user-directed — opening/switching to a session clears its recent-activity indicator).
- **Inactive threshold: user-configurable** with default **15 minutes** (session-settled: user-directed; KTD5).
- **Activity dot may appear on the currently viewed session** when it receives meaningful output (session-settled: user-directed — chosen over “unseen sessions only”). Revisit if noisy in practice.

### Requirements

**List stability**

- R1. Session row order does not change when a background session receives PTY activity unless the user changes sort settings or sessions are created/closed.
- R2. The client updates session rows incrementally (keyed by `session_id`) so polling does not rebuild the entire list DOM when data is unchanged.
- R3. The server returns sessions in a deterministic order for a given sort field (no map-iteration randomness).

**Session metadata (discovery API)**

- R4. `GET /v1/sessions` exposes `created_at` (RFC3339 or Unix ms) for each session.
- R5. The API exposes `last_meaningful_activity_at` (or equivalent) per session, updated when meaningful PTY output occurs.
- R6. The API continues to expose `cli` when the session was created with an agent CLI; otherwise the UI derives a display label from `workspace` basename (e.g. the project folder name for shell sessions).

**Sort**

- R7. The session panel provides a sort dropdown with at least: **time created** (newest/oldest), **agent/workspace label** (alphabetical), **session id** (stable), and **duration** (time since creation, when `created_at` is available).
- R8. Default sort is **newest first** by creation time.
- R9. The user's sort choice persists in browser `localStorage` across reloads.

**Activity status**

- R10. Each session row shows a **status circle** on the left.
- R11. **Recent activity** state: meaningful output since last user acknowledgment — visually distinct (e.g. accent color).
- R12. **Idle** state: user opened the session row since last meaningful activity, or session is active but acknowledged — neutral indicator.
- R13. **Inactive** state: no meaningful activity for longer than the configured threshold — dimmer/muted indicator (distinct from idle).
- R14. Clicking/opening a session row clears recent activity for that session (acknowledge).
- R15. Inactive threshold is configurable in the session panel (or settings affordance within it); default **15 minutes** (KTD5).

**Meaningful activity (product definition)**

- R16. Meaningful activity means new user-visible terminal content (e.g. new tail lines or substantive screen change), not bare cursor moves or ANSI housekeeping alone. Detection: tail-line fingerprint per KTD1.

### User Flows

**F1 — Observe multiple sessions without list jump**

1. User has three sessions open; one background session receives agent output while another is focused.
2. Sidebar order stays fixed (per user's sort).
3. The background session's status dot shows recent activity; no row flicker or reorder.

**F2 — Acknowledge activity**

1. User sees a dot on session B while viewing session A.
2. User clicks session B.
3. Dot clears to idle; terminal attaches to B.

**F3 — Inactive session**

1. Session has no meaningful activity for longer than the configured threshold.
2. Dot shows inactive (muted), even if the session process is still running.

**F4 — Change sort**

1. User opens sort dropdown, selects “Agent / workspace A–Z”.
2. Order updates once; remains stable until sort changes again or sessions added/removed.

### Success Criteria

- SC1. With five concurrent sessions and 2s polling, no visible row flicker when only the active session's terminal updates.
- SC2. With background activity on a non-focused session, list order unchanged and status dot updates within one poll cycle.
- SC3. Sort preference survives browser refresh.
- SC4. Shell sessions display a recognizable workspace label; CLI-spawned sessions display `cli` name when set.

### Scope Boundaries

**In scope**

- Browser session panel (`web/app.js`, `web/style.css`, `web/index.html`)
- `GET /v1/sessions` response enrichment
- Server-side tracking of creation time and last meaningful activity
- Client-side acknowledgment state (per session, per browser tab)

**Out of scope**

- Per-session SSE/WebSocket subscriptions for the sidebar alone
- Full multi-column table layout in the sidebar
- Terminal renderer changes (xterm observe mode, WebGL/DOM)
- Mobile/responsive redesign of the session panel
- Cross-browser sync of acknowledgment state (local to browser unless later specified)
- Session list changes on the headless API beyond fields needed for this feature

### Deferred (product)

- Push notifications or sound on background activity
- Global “mark all read” control
- Server-persisted acknowledgment (shared across browsers)
