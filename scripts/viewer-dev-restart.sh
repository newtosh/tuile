#!/usr/bin/env bash
# Rebuild tuile, restart the local viewer server, and recreate demo sessions.
#
# Usage:
#   ./scripts/viewer-dev-restart.sh           # build + restart + demo sessions
#   ./scripts/viewer-dev-restart.sh --no-build
#   ./scripts/viewer-dev-restart.sh --no-sessions
#
# Env:
#   TUILE_LISTEN   bind address (default 127.0.0.1:7710)
#   TUILE_CONFIG   config path (default tuile.toml in repo root)
#   SHELL          PTY shell for new sessions (default /usr/bin/zsh)

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LISTEN="${TUILE_LISTEN:-127.0.0.1:7710}"
PORT="${LISTEN##*:}"
HOST="${LISTEN%:*}"
BASE="http://${HOST}:${PORT}"
PIDFILE="${ROOT}/.tuile-dev.pid"
LOGFILE="${ROOT}/.tuile-dev.log"
BUILD=1
SESSIONS=1

for arg in "$@"; do
  case "$arg" in
    --no-build) BUILD=0 ;;
    --no-sessions) SESSIONS=0 ;;
    -h|--help)
      sed -n '2,12p' "$0"
      exit 0
      ;;
    *)
      echo "unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

stop_server() {
  if [[ -f "$PIDFILE" ]]; then
    local pid
    pid="$(cat "$PIDFILE")"
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in {1..20}; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.1
      done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$PIDFILE"
  fi

  if command -v fuser >/dev/null 2>&1; then
    fuser -k "${PORT}/tcp" 2>/dev/null || true
  elif command -v lsof >/dev/null 2>&1; then
    local pids
    pids="$(lsof -ti ":${PORT}" 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      kill $pids 2>/dev/null || true
      sleep 0.2
    fi
  fi
}

wait_for_server() {
  local i
  for i in {1..50}; do
    if curl -sf "$BASE/version" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.15
  done
  echo "tuile did not become ready at $BASE" >&2
  if [[ -f "$LOGFILE" ]]; then
    echo "--- server log ---" >&2
    tail -20 "$LOGFILE" >&2 || true
  fi
  return 1
}

cd "$ROOT"

if [[ "$BUILD" -eq 1 ]]; then
  echo "==> building tuile"
  make build
fi

echo "==> stopping existing server on :$PORT"
stop_server
sleep 0.2

echo "==> starting tuile serve ($LISTEN)"
: >"$LOGFILE"
nohup env SHELL="${SHELL:-/usr/bin/zsh}" ./bin/tuile serve --listen "$LISTEN" --force >>"$LOGFILE" 2>&1 &
echo $! >"$PIDFILE"
disown $! 2>/dev/null || true

wait_for_server
echo "==> tuile ready at $BASE"

if [[ "$SESSIONS" -eq 1 ]]; then
  echo "==> creating demo sessions"
  TUILE_BASE="$BASE" SHELL="${SHELL:-/usr/bin/zsh}" "$ROOT/scripts/viewer-demo-sessions.sh"
fi

echo "==> viewer: $BASE/view"
