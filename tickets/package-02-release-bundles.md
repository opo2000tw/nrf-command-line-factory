# package-02: per-platform release bundle 與 checksum

- 階段: Package
- 優先: P1
- 狀態: done

## 背景
`dist/` 目錄未版控，也沒有標準化的交付物格式，目前若要把建置結果交給站台或使用者，缺少一個可直接散布的壓縮包流程。

## 驗收條件
- [x] 每個平台都產出 `nrf-factory-<version>-<os>-<arch>.zip`，內含該平台的執行檔、README、docs/INSTALL-TOOLS.md，以及（mac 平台）exec.command
- [x] 產出對應的 SHA256SUMS 檔案
- [x] 壓縮包不包含韌體檔、nrfutil、J-Link 等外部工具
- [x] 打包步驟納入 build 腳本或 release 流程，不需手動操作

## 相關
- 檔案: dist/, README.md
- 依賴: build-01, release-02
