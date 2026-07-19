---
applyTo: "web/**"
---

When reviewing viewer changes:

- `web/app.js` is large; prefer small, localized edits over refactors unless requested.
- Session list UI must remain a flex scroll container: `.session-list` needs `flex: 1`, `min-height: 0`, and `overflow-y: auto` (guarded by `web/style.test.js`).
- Client session pruning logic lives in `web/session-state.js`; wire through `syncSessions()` in `app.js`.
- Add or update `web/*.test.js` for pure JS helpers and CSS contracts; use chromedp integration tests for DOM behavior.
