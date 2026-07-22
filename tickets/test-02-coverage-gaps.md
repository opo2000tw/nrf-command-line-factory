# test-02: 測試覆蓋缺口

- 階段: Test
- 優先: P2
- 狀態: todo

## 背景
test-01 的 route-mock 套件涵蓋主要 UI 路徑,但仍有數條與硬體、逾時、串流中斷與後端分支相關的路徑未覆蓋。這些是三方審查(見 .docs/reviews/2026-07-23-improvement-review.md)點出的缺口。

## 驗收條件
- [ ] Playwright:多個 J-Link 與零個 J-Link 的錯誤訊息經 handleFlash 整條路徑呈現
- [ ] Playwright:逾時訊息(燒錄逾時 / 安裝逾時)前端呈現
- [ ] Playwright:串流中途錯誤(送出部分 NDJSON 後連線中斷)與 JSON 解析保護
- [ ] Playwright:非 .hex 檔在前端即被擋(不必等後端 400)
- [ ] Go 表格測試:handleReset/handleTool/handleInstallDevice 的 409 分支、各 handler 的 405、handleTool 缺檔與超限
- [ ] Go 表格測試:parseJLinkSerials 全空與全非法輸入
- [ ] 逾時常數改為可注入,補逾時分支測試
- [ ] CI 加 `go test -race` 與並行 flash/reset/tool/install 競態測試
- [ ] browser_test.go 補 clearSingletonLocks 與 findBrowser 直接測試
- [ ] 對 release candidate 做有限度 HIL 驗收並記錄(不進每個 PR)

## 相關
- 檔案: tests/*.spec.ts, cmd/nrf-factory/main_test.go, cmd/nrf-factory/browser_test.go, .github/workflows/ci.yml
- 依賴: test-01
