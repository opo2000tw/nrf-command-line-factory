#!/usr/bin/env bash
# Open URL in Chromium-family --app= mode with a dedicated user-data-dir.
# Closing that app window ends the browser process we started (waitable).
#
# Usage:
#   browser-app.sh http://127.0.0.1:PORT
#   browser-app.sh --wait http://127.0.0.1:PORT
set -euo pipefail

wait_mode=0
url=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --wait) wait_mode=1; shift ;;
    -*)
      echo "unknown flag: $1" >&2
      exit 2
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done

if [[ -z "$url" ]]; then
  echo "usage: $0 [--wait] http://127.0.0.1:PORT" >&2
  exit 2
fi
case "$url" in
  http://127.0.0.1:*|http://localhost:*) ;;
  *)
    echo "usage: $0 [--wait] http://127.0.0.1:PORT" >&2
    exit 2
    ;;
esac

if ! curl -sf --max-time 2 "$url/api/state" >/dev/null 2>&1; then
  echo "服務尚未就緒或已停止，不開視窗: $url" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROFILE_DIR="${NRF_FACTORY_CHROME_PROFILE:-${ROOT}/.run/chrome-profile}"
mkdir -p "$PROFILE_DIR"

find_browser() {
  local c
  for c in google-chrome google-chrome-stable chromium chromium-browser microsoft-edge msedge brave-browser brave; do
    if command -v "$c" >/dev/null 2>&1; then
      command -v "$c"
      return
    fi
  done
  if [[ "$(uname -s)" == "Darwin" ]]; then
    local bin
    for bin in \
      "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
      "/Applications/Chromium.app/Contents/MacOS/Chromium" \
      "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge" \
      "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
    do
      if [[ -x "$bin" || -f "$bin" ]]; then
        echo "$bin"
        return
      fi
    done
  fi
  if [[ -n "${MSYSTEM:-}${WINDIR:-}" ]]; then
    local bin
    for bin in \
      "/c/Program Files/Google/Chrome/Application/chrome.exe" \
      "/c/Program Files (x86)/Google/Chrome/Application/chrome.exe" \
      "/c/Program Files/Microsoft/Edge/Application/msedge.exe" \
      "/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe"
    do
      if [[ -f "$bin" ]]; then
        echo "$bin"
        return
      fi
    done
  fi
  return 1
}

BIN="$(find_browser || true)"
if [[ -z "${BIN:-}" ]]; then
  echo "找不到支援 --app= 的 Chrome / Edge / Chromium / Brave" >&2
  echo "請手動開啟: $url" >&2
  exit 1
fi

# Stale SingletonLock from a killed session can make Chrome exit immediately.
rm -f "${PROFILE_DIR}/SingletonLock" "${PROFILE_DIR}/SingletonCookie" "${PROFILE_DIR}/SingletonSocket" 2>/dev/null || true

# Launch without nohup so we keep a real child PID to wait on.
# Main Chrome process (with dedicated profile + single --app window) exits when the window closes.
"$BIN" \
  --user-data-dir="$PROFILE_DIR" \
  --no-first-run \
  --no-default-browser-check \
  --disable-features=Translate \
  --app="$url" \
  >/dev/null 2>&1 &
chrome_pid=$!

# Brief settle: if Chrome rejects the profile it exits at once.
sleep 0.8
if ! kill -0 "$chrome_pid" 2>/dev/null; then
  echo "瀏覽器立刻結束（pid $chrome_pid）。可刪 .run/chrome-profile 後重試。" >&2
  exit 1
fi

if [[ "$wait_mode" -ne 1 ]]; then
  exit 0
fi

# Block until user closes the app window (main process exits).
wait "$chrome_pid" 2>/dev/null || true
exit 0
