#!/usr/bin/env bash
# Capture README viewer screenshots into docs/images/.
# Requires Chrome/Chromium (chromedp) and a built tuile binary is not required
# (the test starts an in-process server).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
CAPTURE_README=1 go test -tags=integration -run TestCaptureREADMEScreenshots ./test/integration/... -count=1 -timeout 3m
