#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="${TUILE_CONFIG:-$ROOT/tuile.toml}"
BOOTSTRAP="$(grep -E '^bootstrap_secret' "$CONFIG" | sed 's/.*= *"\(.*\)".*/\1/')"
BASE="${TUILE_BASE:-http://127.0.0.1:7710}"

start_session() {
  local label="$1"
  local script="${2:-}"
  CREATED=$(curl -s -X POST "$BASE/v1/sessions" \
    -H "Authorization: Bearer $BOOTSTRAP" \
    -H "Content-Type: application/json" \
    -d '{"workspace":"/tmp","cols":120,"rows":36}')
  SID=$(echo "$CREATED" | jq -r '.session_id')
  TOKEN=$(echo "$CREATED" | jq -r '.token')
  HUMAN=$(curl -s -X POST "$BASE/v1/sessions/$SID/attach" \
    -H "Authorization: Bearer $BOOTSTRAP" \
    -H "Content-Type: application/json" \
    -d '{"mode":"human"}' | jq -r '.token')

  if [[ -n "$script" ]]; then
    python3 - "$SID" "$TOKEN" "$BASE" "$script" <<'PY'
import base64
import json
import sys
import urllib.request

sid, token, base, script = sys.argv[1:5]
b64 = base64.b64encode(script.encode()).decode()
body = json.dumps({"input": f"sleep 0.3; echo {b64} | base64 -d | bash\n"}).encode()
req = urllib.request.Request(
    f"{base}/v1/sessions/{sid}/input",
    data=body,
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    method="POST",
)
urllib.request.urlopen(req).read()
PY
  fi

  echo "$label: $BASE/view?session=$SID&token=$HUMAN"
}

read -r -d '' DEMO_SCRIPT <<'EOF' || true
#!/usr/bin/env bash
export PYENV_SKIP_REHASH=1
export TERM=xterm-256color
cd /tmp
clear
printf "\n\033[1mTuile theme demo\033[0m — switch App appearance + Terminal theme in Settings\n\n"
printf "\033[1mANSI palette:\033[0m\n"
for i in 0 1 2 3 4 5 6 7; do printf " \033[3%sm█\033[0m" "$i"; done
printf "\n"
for i in 0 1 2 3 4 5 6 7; do printf " \033[9%sm█\033[0m" "$i"; done
printf "\n\n"
printf "Nerd icons: \ue718 \uf489 \ue7c8 \uf420\n"
printf "Ligatures: => !== ===\n\n"
printf "256 colors (sample): "
for c in 196 208 226 46 51 99 201 161; do printf "\033[38;5;%sm█\033[0m" "$c"; done
printf "\n\n"
if command -v nvim >/dev/null && [ -f /tmp/tuile-export-readme.md ]; then
  exec nvim /tmp/tuile-export-readme.md
else
  exec "${SHELL:-/bin/sh}" -i
fi
EOF

start_session "Theme demo" "$DEMO_SCRIPT"
start_session "Plain shell" "export PYENV_SKIP_REHASH=1; exec ${SHELL:-/usr/bin/zsh} -i"
