#!/bin/bash
# Double-click this file in Finder (macOS) to run nRF Factory.
# .command files open Terminal automatically; plain .sh usually does not.
cd "$(dirname "$0")" || exit 1
chmod +x ./exec.sh ./scripts/browser-app.sh 2>/dev/null || true
./exec.sh
status=$?
echo ""
if [[ $status -ne 0 ]]; then
  echo "結束代碼: $status"
  echo "按 Return 關閉視窗…"
  read -r _
fi
exit "$status"
