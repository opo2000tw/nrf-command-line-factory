# release-02: GitHub Release 發布流程

- 階段: Release
- 優先: P1
- 狀態: done

## 背景
專案已推送至 github.com/opo2000tw/nrf-command-line-factory 的 main 分支；tag 觸發的自動化發布流程已建立（`.github/workflows/release.yml`），並在 v0.1.0 tag 實跑成功，產出雙平台 zip 與 SHA256SUMS。

## 驗收條件
- [x] push `v*` tag 時觸發 GitHub Actions matrix 建置 windows / darwin
- [x] 建置完成後依 package-02 的規則打包，並產出 SHA256SUMS
- [x] 以 `gh release create` 建立 Release，附上各平台 zip 與 checksums；release notes 用 `--generate-notes` 由 GitHub 自動生成（不讀 CHANGELOG.md，逐版變更仍以 CHANGELOG 為準）
- [x] 產物版本與 tag 一致（串接 build-02 的版本注入）
- [x] 文件化手動 fallback 流程，說明如何用本機 `gh release create` 補發（README「取得執行檔」節）

## 相關
- 檔案: .github/workflows/release.yml, .github/workflows/ci.yml
- 依賴: build-01, build-02, package-02, release-01
