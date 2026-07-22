#!/bin/bash
# Double-click this file in Finder (macOS) to launch nRF Factory.
# Closing the app window (or Ctrl+C in this terminal) stops the service.
cd "$(dirname "$0")" || exit 1

BIN="${NRF_FACTORY_BIN:-}"
if [[ -z "$BIN" ]]; then
  case "$(uname -m)" in
    arm64|aarch64) BIN="./dist/nrf-factory-darwin-arm64" ;;
    *)             BIN="./dist/nrf-factory-darwin-amd64" ;;
  esac
fi

chmod +x "$BIN" 2>/dev/null || true
if [[ ! -x "$BIN" ]]; then
  echo "找不到執行檔: $BIN"
  echo "請先建置 dist/ 產物，或設 NRF_FACTORY_BIN 指向執行檔"
  read -r -p "按 Return 關閉…" _
  exit 1
fi

exec "$BIN"
