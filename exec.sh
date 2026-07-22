#!/usr/bin/env bash
# nRF Factory session launcher.
#
# ./exec.sh lifecycle:
#   start server → open --app= window → wait until browser exits → kill server
#   Closing the app window stops the flash service.
#
#   ./exec.sh | start
#   ./exec.sh open | status | stop
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BROWSER_APP="${ROOT}/scripts/browser-app.sh"
RUN_DIR="${ROOT}/.run"
PID_FILE="${RUN_DIR}/nrf-factory.pid"
URL_FILE="${RUN_DIR}/nrf-factory.url"
LOG_FILE="${RUN_DIR}/nrf-factory.log"
ADDR="${NRF_FACTORY_ADDR:-127.0.0.1:17832}"
PROFILE_DIR="${NRF_FACTORY_CHROME_PROFILE:-${ROOT}/.run/chrome-profile}"

cmd="${1:-start}"
if [[ $# -gt 0 ]]; then
  shift
fi

pick_binary() {
  if [[ -n "${NRF_FACTORY_BIN:-}" ]]; then
    echo "$NRF_FACTORY_BIN"
    return
  fi
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin)
      case "$arch" in
        arm64|aarch64) echo "${ROOT}/dist/nrf-factory-darwin-arm64" ;;
        x86_64|amd64)  echo "${ROOT}/dist/nrf-factory-darwin-amd64" ;;
        *)             echo "${ROOT}/nrf-factory" ;;
      esac
      ;;
    linux) echo "${ROOT}/nrf-factory" ;;
    msys*|mingw*|cygwin*|*_nt-*) echo "${ROOT}/dist/nrf-factory-windows-amd64.exe" ;;
    *)
      if [[ -f "${ROOT}/dist/nrf-factory-windows-amd64.exe" ]]; then
        echo "${ROOT}/dist/nrf-factory-windows-amd64.exe"
      else
        echo "${ROOT}/nrf-factory"
      fi
      ;;
  esac
}

is_pid_alive() {
  local pid="${1:-}"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

is_running() {
  [[ -f "$PID_FILE" ]] || return 1
  is_pid_alive "$(cat "$PID_FILE" 2>/dev/null || true)"
}

read_url() {
  [[ -f "$URL_FILE" ]] || return 1
  tr -d '\r\n' <"$URL_FILE"
}

url_alive() {
  local url="$1"
  [[ -n "$url" ]] && curl -sf --max-time 2 "$url/api/state" >/dev/null 2>&1
}

# Kill browser processes that still reference our dedicated profile.
stop_chrome_profile() {
  local pids
  pids="$(
    ps -axo pid=,command= 2>/dev/null \
      | grep -F -- "--user-data-dir=${PROFILE_DIR}" \
      | grep -Ei 'Google Chrome|Chromium|Microsoft Edge|Brave|chrome|msedge|chromium' \
      | grep -v 'grep' \
      | awk '{print $1}' || true
  )"
  if [[ -z "${pids//[$' \t\n']/}" ]]; then
    return 0
  fi
  # shellcheck disable=SC2086
  kill $pids 2>/dev/null || true
  sleep 0.3
  pids="$(
    ps -axo pid=,command= 2>/dev/null \
      | grep -F -- "--user-data-dir=${PROFILE_DIR}" \
      | grep -Ei 'Google Chrome|Chromium|Microsoft Edge|Brave|chrome|msedge|chromium' \
      | grep -v 'grep' \
      | awk '{print $1}' || true
  )"
  if [[ -n "${pids//[$' \t\n']/}" ]]; then
    # shellcheck disable=SC2086
    kill -9 $pids 2>/dev/null || true
  fi
  rm -f "${PROFILE_DIR}/SingletonLock" "${PROFILE_DIR}/SingletonCookie" "${PROFILE_DIR}/SingletonSocket" 2>/dev/null || true
}

# Kill anything still listening on our fixed factory port (orphans without pid file).
kill_port_listeners() {
  local port pids
  port="${ADDR##*:}"
  [[ -n "$port" ]] || return 0
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)"
    if [[ -n "${pids:-}" ]]; then
      # shellcheck disable=SC2086
      kill $pids 2>/dev/null || true
      sleep 0.2
      # shellcheck disable=SC2086
      kill -9 $pids 2>/dev/null || true
    fi
  fi
}

stop_server() {
  local pid=""
  if [[ -f "$PID_FILE" ]]; then
    pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  fi
  if is_pid_alive "$pid"; then
    echo "停止 nRF Factory (pid $pid)…"
    kill "$pid" 2>/dev/null || true
    for _ in $(seq 1 50); do
      is_pid_alive "$pid" || break
      sleep 0.05
    done
    is_pid_alive "$pid" && kill -9 "$pid" 2>/dev/null || true
  fi
  kill_port_listeners
  rm -f "$PID_FILE" "$URL_FILE"
}

cmd_status() {
  if is_running; then
    local pid url
    pid="$(cat "$PID_FILE")"
    url="$(read_url || true)"
    echo "running  pid=$pid  url=${url:-?}"
    if [[ -n "${url:-}" ]] && url_alive "$url"; then
      echo "http ok"
    else
      echo "http not ready (see $LOG_FILE)"
    fi
    return 0
  fi
  echo "stopped"
  return 1
}

cmd_stop() {
  stop_chrome_profile
  if ! is_running; then
    rm -f "$PID_FILE" "$URL_FILE"
    echo "already stopped"
    return 0
  fi
  stop_server
  echo "stopped"
}

start_server_only() {
  chmod +x "$BROWSER_APP" 2>/dev/null || true

  if is_running; then
    local existing
    existing="$(read_url || true)"
    if [[ -n "${existing:-}" ]] && url_alive "$existing"; then
      echo "$existing"
      return 0
    fi
    echo "pid 檔殘留但服務無回應，先清理…" >&2
    stop_server
  fi

  local bin
  bin="$(pick_binary)"
  if [[ ! -f "$bin" ]]; then
    echo "找不到執行檔: $bin" >&2
    exit 1
  fi
  chmod +x "$bin" 2>/dev/null || true
  mkdir -p "$RUN_DIR"
  : >"$LOG_FILE"
  rm -f "$PID_FILE" "$URL_FILE"

  if curl -sf --max-time 1 "http://${ADDR}/api/state" >/dev/null 2>&1; then
    echo "port 已被占用: http://${ADDR} — 先 ./exec.sh stop" >&2
    exit 1
  fi

  echo "啟動: $bin -addr $ADDR" >&2
  nohup "$bin" -no-browser -addr "$ADDR" >>"$LOG_FILE" 2>&1 &
  local server_pid=$!
  echo "$server_pid" >"$PID_FILE"

  local url="" i
  for i in $(seq 1 100); do
    if ! is_pid_alive "$server_pid"; then
      echo "nRF Factory 啟動失敗:" >&2
      cat "$LOG_FILE" >&2
      rm -f "$PID_FILE" "$URL_FILE"
      exit 1
    fi
    if url="$(grep -Eo 'http://127\.0\.0\.1:[0-9]+' "$LOG_FILE" 2>/dev/null | head -n1)" && [[ -n "$url" ]]; then
      break
    fi
    sleep 0.05
  done
  if [[ -z "$url" ]]; then
    echo "逾時：無 URL" >&2
    cat "$LOG_FILE" >&2
    stop_server
    exit 1
  fi
  for i in $(seq 1 100); do
    if curl -sf --max-time 1 "$url/api/state" >/dev/null 2>&1; then
      printf '%s\n' "$url" >"$URL_FILE"
      echo "$url"
      return 0
    fi
    if ! is_pid_alive "$server_pid"; then
      echo "就緒前結束" >&2
      cat "$LOG_FILE" >&2
      exit 1
    fi
    sleep 0.05
  done
  echo "逾時：/api/state" >&2
  stop_server
  exit 1
}

cmd_open() {
  local url
  url="$(read_url || true)"
  if [[ -z "${url:-}" ]] || ! url_alive "$url"; then
    echo "服務未在跑，請用: ./exec.sh" >&2
    exit 1
  fi
  echo "opening --app=$url"
  "$BROWSER_APP" "$url"
}

cmd_start() {
  local url
  url="$(start_server_only)"

  echo "nRF Factory: $url"
  echo "server pid: $(cat "$PID_FILE")"
  echo "log: $LOG_FILE"
  echo ""
  echo "※ 關掉 app 視窗 = 停止燒錄服務"
  echo "※ Ctrl+C 也會停服務"
  echo ""

  cleanup() {
    echo ""
    echo "清理中…"
    stop_chrome_profile
    stop_server
  }
  trap cleanup INT TERM

  if ! "$BROWSER_APP" --wait "$url"; then
    echo "瀏覽器 session 異常結束" >&2
    cleanup
    trap - INT TERM
    exit 1
  fi

  echo "app 視窗已關閉 → 停止服務"
  stop_server
  # helpers may linger briefly
  stop_chrome_profile
  trap - INT TERM
  echo "done"
}

case "$cmd" in
  start|run|"") cmd_start "$@" ;;
  open|reopen)  cmd_open ;;
  stop|kill)    cmd_stop ;;
  status|st)    cmd_status ;;
  -h|--help|help)
    sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
    ;;
  *)
    echo "unknown command: $cmd (start|open|stop|status)" >&2
    exit 2
    ;;
esac
