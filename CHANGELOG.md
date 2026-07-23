# Changelog

本專案採 SemVer；git tag 慣例 `vMAJOR.MINOR.PATCH`。版號與 bump 規則見 `tickets/release-01-versioning-policy.md`。

## [1.0.0]

首個正式版。內容與 0.1.6 相同，宣告介面與操作流程穩定，0.1.x 系列的功能自此視為正式基線：

- 燒錄流程：外部 .hex 選取、L/R 側別、燒錄與成功計數、清除計數；燒錄期間 UI 鎖定、瀏覽器斷線不中斷硬體寫入、關窗前確認。
- 環境偵測：preflight 回報 nrfutil core / device 版本與 SEGGER J-Link 實際安裝版本；偵測裝置按鈕分別回報 J-Link 與 MCU 存在狀態；一鍵安裝 device 命令（已安裝自動灰化）。
- 交付：release zip 內附 nrfutil（3rd/），解壓即用；zip 佈局鏡射開發樹；SHA256SUMS 與內容驗證納入 release workflow。
- 品質：Go 單元測試與 Playwright UI 功能測試（CI 全跑），macOS 與 Windows（Parallels VM）雙平台實測驗證。

## [0.1.6]

### Fixed
- busy 狀態釋放改為 panic-safe：flash 與 install-device handler 補上與其他 handler 一致的 defer 安全網，避免極端情況下 busy 卡死導致所有操作回 409、需重啟才能恢復。

### Changed
- CI 警告清理（Go cache 與 Node 20 actions）。
- 文件：AGENTS.md 新增 prlctl Windows VM 驗證流程（實測過的 EncodedCommand、共享交換、host 端拍照與各坑）；README 補手動補發 release 流程；tickets 全面稽核校正（package-02/release-02/run-04/package-04/test-01 過時敘述、補交叉引用）。

## [0.1.5]

### Added
- 偵測裝置按鈕：一鍵回報兩個獨立狀態——J-Link 是否連接（`nrfutil device list`）與 MCU 是否存在（`nrfutil device device-info` 實際讀晶片），各自紅綠顯示；無探針或無晶片時給出明確指引，macOS 與 Windows 同流程。
- preflight 顯示 SEGGER J-Link 實際安裝版本（如 `SEGGER J-Link V9.60 OK`）：解析 SEGGER pack 目錄（V960、V924a 形式）、優先採 active install 或最新版；缺 J-Link 或讀不到版本時，訊息附上 nrfutil device 命令的 tested 版（如 V9.24a）指出該裝哪版。

### Changed
- 「安裝 device 命令」按鈕在 device 命令已安裝時灰化並顯示已安裝，不再可重複觸發。

## [0.1.4]

### Fixed
- 燒錄中的 UI 鎖定：先前 flash 送出後 UI 會短暫解鎖，操作員可能誤按清除計數或安裝等按鈕；現在整段燒錄期間確實鎖住，且燒錄中重新整理或關閉視窗會先跳確認。
- 燒錄／安裝不再被瀏覽器斷線中斷：重新整理、關閉視窗或連線瞬斷不會再腰斬正在進行的 nrfutil（避免半寫入的目標）；關閉視窗時若仍在燒錄，會等到完成才停止服務（可再按一次 Ctrl+C 強制停止）。

### Changed
- release 產物加上 SHA256SUMS 回驗與 zip 內容檢查；CI 升到 Node 22 並修正 go.sum 快取警告。

## [0.1.3]

### Fixed
- 後端斷線時的按鈕鎖定：離線（無法連線本機服務）時，除了 flash，清除計數與安裝 device 命令按鈕也一併停用，避免按下去才報錯；先前只有 flash 會停用。

### Added
- Playwright 功能測試套件：以 route mock 涵蓋 UI 的所有按鈕與輸入（firmware/tool 選檔、L/R 選側、燒錄成功/失敗/忙碌/離線、清除計數、安裝 device 命令），不需 nrfutil 或 J-Link 硬體即可執行；CI 新增 ui-test job 一併跑過。

## [0.1.2]

### Added
- 內附 nrfutil：release zip 依平台附上 `3rd/nrfutil`（macOS）/ `3rd/nrfutil.exe`（Windows），程式啟動自動採用（優先序 `NRFUTIL_PATH` > 內附 `3rd/` > PATH），macOS 並自動清除 quarantine 與補上執行權限；工站不必另裝或手動指定 nrfutil core。UI 的 nRF Util 選取項目會標示使用內附版本。

### Fixed
- J-Link 偵測改為同時掃 SEGGER 標準安裝路徑（Windows `C:\Program Files\SEGGER\JLink[_V*]`、macOS `/Applications/SEGGER/JLink*`）與 PATH：安裝 J-Link 後不必手動設 PATH 即可轉綠。缺 J-Link 時的訊息會附上 nrfutil 回報的 tested 版，指出該裝哪一版。

## [0.1.1]

### Fixed
- macOS 雙擊啟動：release zip 佈局改為鏡射開發樹（執行檔在 `dist/`），修正舊平整 zip 導致啟動器找不到執行檔的問題。

### Changed
- 移除 `exec.command`：執行檔本身即為雙擊啟動目標（macOS 由 Finder 以終端機執行 Unix 執行檔）。
- release zip 只附兩份文件：`CHANGELOG.md`（版本演進）與 `docs/INSTALL-AND-RUN.md`（安裝 toolchain 與執行）；`docs/INSTALL-TOOLS.md` 內容併入 INSTALL-AND-RUN 後移除。
- 文件修正：AGENTS.md 專案結構與建置指令對齊實際 flat 佈局。

## [0.1.0]

### Added
- Go 內建 browser app-window session：偵測 Chrome / Edge / Chromium / Brave，以獨立 profile 的 `--app` 視窗開啟；關窗或 Ctrl+C 即停止服務。
- App 內指定 nrfutil 執行檔（`/api/tool`，同 `.hex` 選檔 UX）與一鍵安裝 device 命令（`/api/install-device`，串流輸出到終端）。
- 版本戳記：`-version` 旗標、console 啟動 banner 與 UI footer 顯示版本（`-ldflags -X` 注入）。
- 多平台建置與打包 `Makefile`（`build` / `dist` / `package`），產出 per-platform zip 與 `SHA256SUMS`。
- GitHub Actions：CI（gofmt / vet / test / cross build）與 tag 觸發的 Release 流程。
- 文件：nRF Util 元件與版本管理、Windows / Parallels 驗證紀錄、build→run pipeline tickets。

### Changed
- 啟動改為單一自足執行檔：移除 `exec.sh` / `exec.bat` / `scripts/browser-app.sh`；README 改為「雙擊平台 exe」。
- J-Link 版本以 `nrfutil device --version` 報的 tested 版為準（現 V9.24a），不再以某台機器現裝版為基準。
- 移除 macOS Intel（darwin/amd64）目標；只出 Windows amd64 與 macOS Apple Silicon。
