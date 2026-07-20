#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

echo "==> export package: custom background tests"
go test ./internal/export/... -run 'Custom|Background' -count=1

echo "==> API export: multipart custom background"
go test ./internal/api/... -run MultipartBackground -count=1

echo "==> viewer export options: custom background"
cd web && node --test export-options.test.js

echo "OK: custom background export workflow passed"
