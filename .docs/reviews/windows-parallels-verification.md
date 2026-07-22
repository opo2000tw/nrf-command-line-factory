# Windows / Parallels 驗證紀錄

日期：2026-07-22
對象：把 browser session 生命週期搬進 Go 主程式後（`cmd/nrf-factory/browser.go` + `main.go`），在真 Windows 上的行為驗證。
方式：純 CLI，透過 Parallels Desktop 的 `prlctl` 驅動一台 Windows 11 VM，不手動操作 GUI。

## 環境

| 項目 | 值 |
|------|-----|
| Parallels | `prlctl` 26.4.0 (57513) |
| VM | Windows 11，build 10.0.26100 |
| Guest shell | Windows PowerShell 5.1.26100（Win10/11 內建版；刻意不用 pwsh 7 以顧及 Win10 相容） |
| 受測產物 | `GOOS=windows GOARCH=amd64` 交叉編譯的 `nrf-factory-windows-amd64.exe`（PE32+ console x86-64） |
| nrfutil / J-Link | VM 未安裝（預期 `toolReady:false`） |

## 方法

- 進 VM 執行：`prlctl exec "Windows 11" powershell -EncodedCommand <UTF-16LE base64>`。用 `-EncodedCommand` 避開跨層（zsh → prlctl → PowerShell）引用與 CP950 codepage 問題。
- 進度與結果寫 guest 本機檔（`C:\Users\Public\...`），必要時用秒回的 `type` 拉回 Mac 判讀；`Invoke-WebRequest` 一律加 `-UseBasicParsing`、關 `$ProgressPreference` 以免 CLIXML 雜訊。
- 截圖用 `prlctl capture "Windows 11" --file <path>`：這是 host 端直接取 VM 的虛擬顯示 framebuffer，不需 macOS 螢幕錄製權限，比截 Parallels 視窗可靠。
- exe 從網路分享複製到 guest 後先 `Unblock-File` 去掉 Mark-of-the-Web，避免 SmartScreen 攔。

## 結果

| 驗證項 | 結果 |
|--------|------|
| 交叉編譯產物可在 Windows 執行 | 通過 |
| 綁 loopback、`/api/state` 回應 | 通過，回 JSON，`toolReady:false`（VM 無 nrfutil，符合 preflight 設計） |
| `findBrowser` 在 Windows 選中的路徑 | `C:\Program Files\Google\Chrome\Application\chrome.exe` |
| app 視窗啟動 | Chrome 以 `--app=http://127.0.0.1:<port>` 啟動 |
| SmartScreen / MOTW | `Unblock-File` 後可直接執行 |
| 進程可被終止 | 通過 |

關鍵佐證：成功啟動的那次，`/api/state` 回應與帶 `--app=` 的 Chrome 程序**同時存在**。這代表 exe 的 `cmd.Wait()` 正確地卡在存活的 Chrome master 上，沒有 Windows 常見的「母程序 fork 後秒退」問題（獨立 `--user-data-dir` 讓我們啟動的程序本身就是 browser master）。這正是「關閉視窗即結束服務」的前半段。

## 視覺與生命週期確認（互動 session）

在 VM 內以互動 session 直接雙擊 exe（`\\Mac\Home\nrf-factory-win-test.exe`）後，用 `prlctl capture` 取得畫面：

- app 視窗以 Chrome `--app` 正常開啟並渲染 UI：標題列、無網址列的獨立視窗；紅點狀態、L/R 計數、選 .hex、開始燒錄／清除計數、燒錄軌跡「系統已啟動，等待韌體。」。
- console 印出新的生命週期訊息：`nRF Factory: http://127.0.0.1:<port>` 與「關閉 app 視窗 或按 Ctrl+C 即停止 nRF Factory」。
- 紅點「找不到 nrfutil…」為 VM 未安裝 Nordic 工具的預期結果，不阻擋 UI。
- 關窗→退出：關閉該 app 視窗後，exe 進程消失（process 檢查為不存在），即 `cmd.Wait()` 返回 → server graceful shutdown → 程式退出。

備註：之所以改用互動雙擊，是因為 `prlctl exec` 在 service context（session 0）起的 GUI 不落在可見 console session（同一原因也讓 app 視窗模式的同步 `prlctl exec` 不返回）；`schtasks` 互動 token 導向 console 亦未穩定成功。這些是 Windows session 隔離造成的測試路徑限制，與受測程式無關。

## 對應的既有覆蓋（後半段生命週期）

「關窗 → shutdown → 退出」與上述已確認的 `cmd.Wait()` 阻塞是同一段 Go 碼，且另有覆蓋：

- Host（macOS）smoke：`-no-browser` 起服務 → `curl /api/state` → `SIGINT` → 印「收到 interrupt，停止服務」→ 退出碼 0。
- Unit test（`browser_test.go`）：用 `/bin/sh` fake browser 驗 `startAppWindow` 的 wait（視窗程序結束後 `done` 關閉）與 `kill`（強殺後 `done` 關閉），不綁真 Chrome。

## 待辦與建議

- console subsystem 目前仍是 console exe，雙擊會有 console 視窗；若工廠在意，第二步再改 `-ldflags -H windowsgui`。
- 要讓狀態轉綠並做無裝置 dry-run：在 VM 安裝 nRF Util 並執行 `nrfutil install device`（缺 J-Link 時燒錄會在 `device list` 階段以錯誤中止，屬預期）。
