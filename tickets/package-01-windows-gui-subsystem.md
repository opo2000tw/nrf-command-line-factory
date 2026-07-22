# package-01: Windows GUI subsystem 消除雙擊黑窗

- 階段: Package
- 優先: P2
- 狀態: todo

## 背景
目前產出的是 console 模式 exe，在檔案總管雙擊執行時會閃過或留下一個黑色的 console 視窗（已在 VM 上實測確認），對工廠站台的雙擊使用體驗不佳。

## 驗收條件
- [ ] 提供以 `-ldflags "-H windowsgui"` 建置的 GUI 變體，雙擊執行不再跳出 console 視窗
- [ ] 保留現有 console 變體供除錯使用
- [ ] 決定產線預設要交付哪一種變體
- [ ] 確認 GUI 變體仍能正常開啟 app 視窗，關閉視窗即停止程式，且沒有 console 附著時 stdout 輸出不會導致 panic

## 相關
- 檔案: cmd/nrf-factory
- 依賴: build-01（此為 browser session commit 標記的第二步）
