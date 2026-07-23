#!/usr/bin/env bash
# Prepare a clean SideProject demo session: workspace, URLs, agent prompt.
#
# Usage:
#   ./scripts/sideproject-demo-setup.sh [--cli codex|claude|cursor-cli|...] [--shell]
#   ./scripts/sideproject-demo-setup.sh --prune-only
#
# Env:
#   TUILE_BASE     server URL (default http://127.0.0.1:7710)
#   TUILE_CONFIG   tuile.toml path (default repo root tuile.toml)
#   DEMO_WORKSPACE workspace path (default /tmp/tuile-demo)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="${TUILE_CONFIG:-$ROOT/tuile.toml}"
BASE="${TUILE_BASE:-http://127.0.0.1:7710}"
WORKSPACE="${DEMO_WORKSPACE:-/tmp/tuile-demo}"
PROMPT_FILE="$ROOT/scripts/sideproject-demo-agent-prompt.txt"
CLI=""
SHELL_ONLY=0
PRUNE_ONLY=0

usage() {
  sed -n '2,12p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cli)
      CLI="${2:-}"
      shift 2
      ;;
    --shell)
      SHELL_ONLY=1
      shift
      ;;
    --prune-only)
      PRUNE_ONLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ ! -f "$CONFIG" ]]; then
  echo "missing config: $CONFIG (cp tuile.toml.example tuile.toml)" >&2
  exit 1
fi

BOOTSTRAP="$(grep -E '^bootstrap_secret' "$CONFIG" | sed 's/.*= *"\(.*\)".*/\1/')"
if [[ -z "$BOOTSTRAP" || "$BOOTSTRAP" == "change-me-to-a-long-random-string" ]]; then
  echo "set a real bootstrap_secret in $CONFIG before recording" >&2
  exit 1
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq

if ! curl -sf "$BASE/health" >/dev/null; then
  echo "tuile is not running at $BASE — start with: tuile serve --force" >&2
  exit 1
fi

VER="$(curl -sf "$BASE/version" | jq -r '.version // empty')"
if [[ "$VER" != "v0.3.0" ]]; then
  echo "warning: server reports $VER (expected v0.3.0 for viewer badge)" >&2
fi

prune_sessions() {
  curl -sf -X POST "$BASE/v1/sessions/prune" \
    -H "Authorization: Bearer $BOOTSTRAP" \
    -H "Content-Type: application/json" \
    -d '{"except":[]}' >/dev/null || true
}

prune_sessions

if [[ "$PRUNE_ONLY" -eq 1 ]]; then
  echo "pruned all sessions at $BASE"
  exit 0
fi

mkdir -p "$WORKSPACE"
rm -rf "${WORKSPACE:?}"/*
printf 'demo>\n' >"$WORKSPACE/.tuile-demo-marker"

if [[ "$SHELL_ONLY" -eq 1 ]]; then
  CREATE_BODY=$(jq -n --arg ws "$WORKSPACE" '{workspace:$ws, cols:110, rows:32}')
else
  if [[ -z "$CLI" ]]; then
    if command -v codex >/dev/null 2>&1; then
      CLI=codex
    elif command -v claude >/dev/null 2>&1; then
      CLI=claude
    else
      echo "no agent CLI in PATH; use --shell or --cli <name>" >&2
      exit 1
    fi
  fi
  CREATE_BODY=$(jq -n --arg ws "$WORKSPACE" --arg cli "$CLI" '{workspace:$ws, cli:$cli, cols:110, rows:32}')
fi

CREATED=$(curl -sf -X POST "$BASE/v1/sessions" \
  -H "Authorization: Bearer $BOOTSTRAP" \
  -H "Content-Type: application/json" \
  -d "$CREATE_BODY")
SID=$(echo "$CREATED" | jq -r '.session_id')
AGENT_TOKEN=$(echo "$CREATED" | jq -r '.token')

ATTACHED=$(curl -sf -X POST "$BASE/v1/sessions/$SID/attach" \
  -H "Authorization: Bearer $BOOTSTRAP" \
  -H "Content-Type: application/json" \
  -d '{"mode":"human"}')
HUMAN_TOKEN=$(echo "$ATTACHED" | jq -r '.token')

CONTROL_URL="$BASE/view?session=$SID&token=$AGENT_TOKEN"
OBSERVE_URL="$BASE/view?session=$SID&token=$HUMAN_TOKEN"

cat <<EOF

== SideProject demo session ready ==

Workspace:  $WORKSPACE
Session:    $SID
Server:     $BASE ($VER)

Window layout (split-screen capture):
  LEFT  (control / agent):  $CONTROL_URL
  RIGHT (observe viewer):   $OBSERVE_URL

Before recording:
  1. Open both URLs in separate browser windows (not tabs).
  2. LEFT: click Takeover if not already in control mode.
  3. RIGHT: stay in observe mode; hide sidebar if it clutters the frame.
  4. Settings → font size ≥ 16px on both windows; match terminal theme.
  5. Confirm badge shows $VER and no home paths / secrets on screen.

Agent prompt (paste into LEFT window after CLI starts):
--- copy below ---
$(cat "$PROMPT_FILE")
--- end ---

Beat sheet: docs/plans/2026-07-21-001-feat-sideproject-demo-share-plan.md
Post-process: ./scripts/sideproject-demo-postprocess.sh raw.mkv demo.mp4

EOF

if [[ -n "$CLI" && "$SHELL_ONLY" -eq 0 ]]; then
  echo "CLI spawned in session: $CLI"
fi
