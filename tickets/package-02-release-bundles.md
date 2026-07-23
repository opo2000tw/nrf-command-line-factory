# package-02: per-platform release bundle 與 checksum

- 階段: Package
- 優先: P1
- 狀態: done

## 背景
`dist/` 目錄未版控，也沒有標準化的交付物格式，目前若要把建置結果交給站台或使用者，缺少一個可直接散布的壓縮包流程。

## 驗收條件
- [x] 每個平台都產出 `nrf-factory-<version>-<os>-<arch>.zip`，內含 `dist/` 執行檔、CHANGELOG.md、docs/INSTALL-AND-RUN.md
- [x] 產出對應的 SHA256SUMS 檔案
- [x] 壓縮包不包含韌體檔與 J-Link；nrfutil 自 v0.1.2 起改為內附（`3rd/`，Makefile package 打入 zip），原「不含 nrfutil」條件已被此決策推翻，授權評估見 release-03
- [x] 打包步驟納入 build 腳本或 release 流程，不需手動操作

## 相關
- 檔案: dist/, README.md
- 依賴: build-01, release-02
