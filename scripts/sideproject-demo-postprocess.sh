#!/usr/bin/env bash
# Normalize a screen capture for r/SideProject (silent H.264 MP4).
#
# Usage:
#   ./scripts/sideproject-demo-postprocess.sh input.mkv [output.mp4]
#   ./scripts/sideproject-demo-postprocess.sh input.mp4 demo.mp4 --trim 3:87
#
# Options:
#   --trim START:END   seconds (defaults: 0:90)
#   --width W          scale width (default 1280)
#   --crf N            x264 quality (default 23; lower = better)

set -euo pipefail

INPUT=""
OUTPUT=""
TRIM="0:90"
WIDTH=1280
CRF=23

usage() {
  sed -n '2,10p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --trim)
      TRIM="${2:-}"
      shift 2
      ;;
    --width)
      WIDTH="${2:-}"
      shift 2
      ;;
    --crf)
      CRF="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
    *)
      if [[ -z "$INPUT" ]]; then
        INPUT="$1"
      elif [[ -z "$OUTPUT" ]]; then
        OUTPUT="$1"
      else
        echo "too many arguments" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

if [[ -z "$INPUT" ]]; then
  usage >&2
  exit 1
fi
if [[ -z "$OUTPUT" ]]; then
  base="${INPUT%.*}"
  OUTPUT="${base}-reddit.mp4"
fi

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "ffmpeg not found" >&2
  exit 1
fi

START="${TRIM%%:*}"
END="${TRIM##*:}"
DURATION="$(awk "BEGIN { print $END - $START }")"

echo "==> trim ${START}s + ${DURATION}s → ${OUTPUT} (${WIDTH}px wide, crf ${CRF})"

ffmpeg -y -hide_banner -loglevel warning \
  -ss "$START" -i "$INPUT" -t "$DURATION" \
  -vf "scale=${WIDTH}:-2:flags=lanczos,format=yuv420p" \
  -an \
  -c:v libx264 -preset medium -crf "$CRF" -movflags +faststart \
  "$OUTPUT"

echo "==> done: $OUTPUT"
echo "    upload this file directly to Reddit (v.redd.it); keep under ~90s and <1GB"
