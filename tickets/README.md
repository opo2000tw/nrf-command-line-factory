# Tickets 索引

以下依 Build / Package / Release / Run 四個階段列出目前規劃的工作項目，狀態欄位反映各 ticket 檔案內的 `狀態` 欄位。

## Build

| ID | 標題 | 優先 | 狀態 |
| --- | --- | --- | --- |
| [build-01](build-01-reproducible-build.md) | 可重現的多平台建置腳本 | P1 | done |
| [build-02](build-02-version-stamping.md) | 版本戳記嵌入執行檔 | P1 | done |

## Package

| ID | 標題 | 優先 | 狀態 |
| --- | --- | --- | --- |
| [package-01](package-01-windows-gui-subsystem.md) | Windows GUI subsystem 消除雙擊黑窗 | P2 | todo |
| [package-02](package-02-release-bundles.md) | per-platform release bundle 與 checksum | P1 | done |
| [package-03](package-03-macos-gatekeeper.md) | macOS Gatekeeper / 簽章策略 | P3 | todo |
| [package-04](package-04-windows-signing.md) | Windows 程式碼簽章 / SmartScreen | P2 | todo |

## Release（GitHub / 版號）

| ID | 標題 | 優先 | 狀態 |
| --- | --- | --- | --- |
| [release-01](release-01-versioning-policy.md) | 版號策略（SemVer + tag + CHANGELOG） | P1 | done |
| [release-02](release-02-github-release.md) | GitHub Release 發布流程 | P1 | done |

## Run

| ID | 標題 | 優先 | 狀態 |
| --- | --- | --- | --- |
| [run-01](run-01-preflight-check-mode.md) | 站台 preflight -check 模式 | P2 | todo |
| [run-02](run-02-fixed-port-single-instance.md) | 固定埠 single-instance 友善處理 | P3 | todo |
| [run-03](run-03-persist-nrfutil-path.md) | nrfutil 路徑持久化 | P3 | todo |
| [run-04](run-04-windows-console-close.md) | Windows 關 console 的 graceful shutdown | P3 | todo |
