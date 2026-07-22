# Changelog

本專案採 SemVer；git tag 慣例 `vMAJOR.MINOR.PATCH`。版號與 bump 規則見 `tickets/release-01-versioning-policy.md`。

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
