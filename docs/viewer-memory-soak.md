# Viewer memory soak verification

Manual recipe to confirm the Tuile browser viewer (`/view`) does not grow client memory without bound when many PTY sessions exist.

## Prerequisites

- Tuile built locally (`make build`)
- `tuile.toml` with bootstrap secret
- Chrome or Chromium
- Optional: script to create many sessions (`tuile session start <workspace>` in separate terminals)

## Quick client-state check

1. Start the server: `./bin/tuile serve --force`
2. Open `http://127.0.0.1:7710/view?debug=memory` (adds console logs for session count, `sessionCache` size, and ack-map size on each list sync).
3. Create several sessions, then close some server-side.
4. In DevTools → Console, confirm `cache` and `ack` counts track the **current** session list, not historical totals.

## Soak scenario A — many listed, one attached

1. Create 10–20 sessions (different workspaces or repeated `tuile session start /tmp`).
2. Open `/view` with bootstrap secret saved.
3. Attach to one session; leave the tab open for 30 minutes.
4. Take heap snapshots at T+0, T+5m, T+30m (Memory tab → Heap snapshot).
5. **Pass:** heap size plateaus after initial load; no steady linear climb.

## Soak scenario B — attach/switch churn

1. Create 5 sessions.
2. Open `/view`, click each session row 20+ times (switch attach target).
3. Take snapshots at start and after 15 minutes.
4. **Pass:** heap after churn is within ~20% of early plateau, not multiples higher.

## Soak scenario C — long output on one session

1. Attach to a session running continuous output (e.g. `yes` or a noisy agent).
2. Wait 30 minutes.
3. **Pass:** memory rises initially then flattens; xterm scrollback stays bounded (`scrollback: 5000` in `web/app.js`).

## Reading Chrome DevTools Memory

- Prefer **Heap snapshot** over Performance monitor for before/after comparison.
- Compare **JS heap size** and count of **Detached DOM tree** nodes between snapshots.
- Local dev target: tens of MB total for a single viewer tab after plateau — not hundreds of MB climbing over an afternoon without new sessions.

## Automated checks

```bash
make test-web
go test -tags=integration ./test/integration/ -run TestViewerPrunesStaleAckState -count=1
```

These verify ack-map pruning; they do not replace the manual soak above for heap plateau (R7–R8).
