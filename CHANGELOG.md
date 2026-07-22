# Changelog

本專案採 SemVer；git tag 慣例 `vMAJOR.MINOR.PATCH`。版號與 bump 規則見 `tickets/release-01-versioning-policy.md`。

## [Unreleased]

### Added
- Go 內建 browser app-window session：偵測 Chrome / Edge / Chromium / Brave，以獨立 profile 的 `--app` 視窗開啟；關窗或 Ctrl+C 即停止服務。
- App 內指定 nrfutil 執行檔（`/api/tool`，同 `.hex` 選檔 UX）與一鍵安裝 device 命令（`/api/install-device`，串流輸出到終端）。
- 版本戳記：`-version` 旗標、console 啟動 banner 與 UI footer 顯示版本（`-ldflags -X` 注入）。
- 多平台建置與打包 `Makefile`（`build` / `dist` / `package`），產出 per-platform zip 與 `SHA256SUMS`。
- GitHub Actions：CI（gofmt / vet / test / cross build）與 tag 觸發的 Release 流程。
- 文件：nRF Util 元件與版本管理、Windows / Parallels 驗證紀錄、build→run pipeline tickets。

### Changed
- 啟動改為單一自足執行檔：移除 `exec.sh` / `exec.bat` / `scripts/browser-app.sh` 與 `exec.command`；macOS 直接雙擊執行檔（Finder 以終端機執行 Unix 執行檔），README 改為「雙擊平台 exe」。
- J-Link 版本以 `nrfutil device --version` 報的 tested 版為準（現 V9.24a），不再以某台機器現裝版為基準。
