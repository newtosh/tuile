#!/usr/bin/env bash
# Headed Playwright loop for control-mode keyboard input (replaces manual browser testing).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

make build

if ! curl -sf http://127.0.0.1:7710/health >/dev/null 2>&1; then
  echo "Starting tuile on 127.0.0.1:7710..."
  fuser -k 7710/tcp 2>/dev/null || true
  sleep 0.5
  ./bin/tuile serve --listen 127.0.0.1:7710 --force >/tmp/tuile-test-control-input.log 2>&1 &
  for _ in $(seq 1 40); do
    if curl -sf http://127.0.0.1:7710/health >/dev/null 2>&1; then
      break
    fi
    sleep 0.25
  done
  if ! curl -sf http://127.0.0.1:7710/health >/dev/null 2>&1; then
    echo "tuile failed to start; see /tmp/tuile-test-control-input.log" >&2
    exit 1
  fi
fi

SETUP_OUT="$(./scripts/sideproject-demo-setup.sh --shell 2>&1)"
echo "$SETUP_OUT"

export TUILE_CONTROL_URL="$(echo "$SETUP_OUT" | sed -n 's/^  LEFT  (control \/ agent):  //p' | head -1)"
if [[ -z "$TUILE_CONTROL_URL" ]]; then
  echo "could not parse control URL from demo setup" >&2
  exit 1
fi

echo "Control URL: $TUILE_CONTROL_URL"

# Install browser once if missing (no-op when already installed).
BROWSER_DIR="$ROOT/test/browser"
if [[ ! -d "$BROWSER_DIR/node_modules/playwright" ]]; then
  echo "Installing Playwright in $BROWSER_DIR..."
  (cd "$BROWSER_DIR" && npm install --no-fund --no-audit)
fi
(cd "$BROWSER_DIR" && npx playwright install chromium) >/dev/null 2>&1 || true

export HEADLESS="${HEADLESS:-0}"
export CHROME_PATH="${CHROME_PATH:-/usr/bin/chromium}"

NODE_PATH="$BROWSER_DIR/node_modules" node scripts/test-control-input-playwright.cjs

# Also run Go integration tests when Chrome/chromedp works (skips otherwise).
go test -tags=integration -count=1 -timeout 120s -run 'TestViewerControlInput' ./test/integration/...
