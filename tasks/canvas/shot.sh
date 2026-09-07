#!/usr/bin/env bash
# Screenshot (or dump the DOM of) a fixture in the canvas gallery.
#
#   tasks/canvas/shot.sh step                    -> /tmp/shot-step.png
#   tasks/canvas/shot.sh wide --out /tmp/w.png
#   tasks/canvas/shot.sh loop40 --dom            -> rendered HTML on stdout
#   tasks/canvas/shot.sh --url http://localhost:5173/run?runID=x
#
# Two things this exists to prevent, both of which have already cost a session:
#   - a hardcoded /nix/store path, which Nix garbage-collected out from under
#     three agent files (see the GC root below);
#   - a silent failure that leaves a stale PNG on disk, so the next `Read` shows
#     the previous change and everything looks fine. Every exit here is loud.
set -uo pipefail

FIXTURE=""; URL=""; OUT=""; MODE=screenshot; SIZE="1600,1100"; BUDGET=15000
while [ $# -gt 0 ]; do
  case "$1" in
    --dom)   MODE=dom;      shift ;;
    --out)   OUT="$2";      shift 2 ;;
    --url)   URL="$2";      shift 2 ;;
    --size)  SIZE="$2";     shift 2 ;;
    --budget) BUDGET="$2";  shift 2 ;;
    -*)      echo "shot: unknown flag $1" >&2; exit 2 ;;
    *)       FIXTURE="$1";  shift ;;
  esac
done

# ---- the browser ---------------------------------------------------------
# Resolved through a GC root, never a literal store path. `nix build` both
# creates the symlink and registers it as an indirect root, so the binary
# cannot be collected while this link exists. If someone deletes the link we
# rebuild it here rather than failing, and it is a cache hit.
ROOT="$HOME/.cache/canvas-shot/chromium"
if [ ! -x "$ROOT/bin/chromium" ]; then
  echo "shot: chromium missing, fetching..." >&2
  mkdir -p "$(dirname "$ROOT")"
  nix build --out-link "$ROOT" nixpkgs#ungoogled-chromium >&2 || {
    echo "shot: could not fetch chromium." >&2; exit 1; }
fi
CHROME="$ROOT/bin/chromium"

# ---- the gallery ---------------------------------------------------------
# Vite takes the first free port from 5173, so the port moves between sessions
# and has been wrong in the docs more than once. Find it, don't assume it.
if [ -z "$URL" ]; then
  PORT=""
  for p in 5173 5174 5175 5176 5177 5178; do
    if [ "$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://localhost:$p/canvas-gallery")" = "200" ]; then
      PORT=$p; break
    fi
  done
  [ -n "$PORT" ] || { echo "shot: no gallery on 5173-5178. Start it with:
  cd ui && pnpm run --filter @inngest/dev-server-ui dev:vite" >&2; exit 1; }
  [ -n "$FIXTURE" ] || { echo "shot: give a fixture id or --url" >&2; exit 2; }
  URL="http://localhost:$PORT/canvas-gallery?fixture=$FIXTURE"
fi

# A URL that does not answer still produces a PNG — chromium renders its own
# error page, and that image read back as evidence is worse than no image.
# Reachability means different things per scheme, so check the right one.
case "$URL" in
  file://*)
    F="${URL#file://}"
    [ -r "$F" ] || { echo "shot: $F is not readable, refusing to screenshot it" >&2; exit 1; } ;;
  *)
    CODE=$(curl -s -o /dev/null -m 5 -w "%{http_code}" "$URL")
    [ "$CODE" = "200" ] || { echo "shot: $URL answered $CODE, refusing to screenshot it" >&2; exit 1; } ;;
esac

COMMON=(--headless --no-sandbox --disable-gpu --hide-scrollbars
        --force-device-scale-factor=1 --window-size="$SIZE"
        --virtual-time-budget="$BUDGET")

if [ "$MODE" = dom ]; then
  "$CHROME" "${COMMON[@]}" --dump-dom "$URL" 2>/dev/null
  exit $?
fi

OUT="${OUT:-/tmp/shot-${FIXTURE:-url}.png}"
# Delete first: chromium leaves the old file in place when it fails, and a
# stale image read back as evidence is the exact failure lessons.md #18 is about.
rm -f "$OUT"
"$CHROME" "${COMMON[@]}" --screenshot="$OUT" "$URL" 2>&1 | grep -v 'ERROR:net/cert' >&2
if [ ! -s "$OUT" ]; then
  echo "shot: no image written for $URL" >&2; exit 1
fi
echo "$OUT"
