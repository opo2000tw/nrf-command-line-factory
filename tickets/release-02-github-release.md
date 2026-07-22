# release-02: GitHub Release 發布流程

- 階段: Release
- 優先: P1
- 狀態: todo

## 背景
專案已推送至 github.com/opo2000tw/nrf-command-line-factory 的 main 分支，但目前完全沒有自動化的發布流程。

## 驗收條件
- [ ] push `v*` tag 時觸發 GitHub Actions matrix 建置 windows / darwin
- [ ] 建置完成後依 package-02 的規則打包，並產出 SHA256SUMS
- [ ] 以 `gh release` 或對應 action 建立 Release，附上各平台 zip、checksums，以及取自 CHANGELOG 的 release notes
- [ ] 產物版本與 tag 一致（串接 build-02 的版本注入）
- [ ] 文件化手動 fallback 流程，說明如何用本機 `gh release create` 補發

## 相關
- 檔案: .github/workflows/（待建）
- 依賴: build-01, build-02, package-02, release-01
