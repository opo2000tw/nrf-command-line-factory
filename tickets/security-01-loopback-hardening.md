# security-01: loopback server 與工具上傳的安全強化

- 階段: Security
- 優先: P1
- 狀態: todo

## 背景
本工具在 127.0.0.1 開一個無認證的本機 HTTP server,並提供上傳可執行檔的端點。三方審查(見 .docs/reviews/2026-07-23-improvement-review.md)指出數個本機攻擊面:惡意網頁可經 CSRF/DNS-rebinding 觸發變更型 API,上傳的檔案會被 chmod 後直接執行,暫存工具路徑固定且可被 symlink 置換。loopback 綁定與隨機埠不等於認證。

## 驗收條件
- [ ] 每次啟動產生高熵 session token;所有變更型 API 驗證 token、精確 Origin 與精確 Host,拒絕 cross-site 與 Origin: null
- [ ] Host 必須等於實際的 127.0.0.1:<port>,防 DNS rebinding
- [ ] /api/tool 不再無條件執行上傳檔:改為隨包已驗證工具,或僅接受簽章 / allowlisted SHA-256,驗證通過前不執行
- [ ] 暫存工具改用 app 私有 0700 隨機目錄 + 隨機檔名 + O_EXCL,執行前確認為 regular file、owner 與 digest
- [ ] NRFUTIL_PATH override 限定 developer mode 並在 UI 顯示非生產警告;正式版預設只接受已驗證 digest 的絕對路徑,找不到即 fail closed
- [ ] 生產綁定 J-Link fixture serial allowlist;program 前做 identity preflight,不退回「任選唯一裝置」
- [ ] CSP 補 frame-ancestors none、object-src none、base-uri none;所有外部文字只用 textContent

## 相關
- 檔案: cmd/nrf-factory/main.go
- 依賴: -
