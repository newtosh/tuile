#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

echo "==> web: svg raster verification helpers"
(cd web && node --test export-svg-verify.test.js)

echo "==> API export: svg endpoint"
go test ./internal/api/... -run TestSessionExportSVG -count=1

if command -v rsvg-convert >/dev/null 2>&1 && command -v python3 >/dev/null 2>&1; then
  for svg in "$@"; do
    if [[ -f "$svg" ]]; then
      echo "==> pixel audit: $svg"
      python3 "$root/scripts/audit-export-svg.py" "$svg"
    fi
  done
else
  echo "note: rsvg-convert/python3 not available; skipping optional pixel audit of local files"
fi

if [[ "${RUN_BROWSER_SVG_TEST:-}" == "1" ]]; then
  echo "==> browser: export svg alignment"
  go test -tags=integration ./test/integration/... -run TestBrowserExportSVGAlignsWithPNG -count=1
fi

echo "OK: svg export verification passed"
