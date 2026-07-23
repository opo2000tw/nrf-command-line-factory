# run-04: Windows 關 console 的 graceful shutdown

- 階段: Run
- 優先: P3
- 狀態: todo

## 背景
signal 處理現已接 `os.Interrupt` 與 `syscall.SIGTERM`，並有 busy-aware graceful shutdown（燒錄中會等待完成、逾時上限、二次訊號強制停止）。但 Windows 上關閉 console 視窗觸發的是 `CTRL_CLOSE_EVENT`，Go 預設不會轉成上述 signal 路徑（無 `SetConsoleCtrlHandler` 專屬處理），收尾仍是 best-effort，可能來不及完成 server shutdown 或 kill 瀏覽器程序。

## 驗收條件
- [ ] 確認並盡量處理 Windows console close 事件下的 graceful shutdown，涵蓋 server shutdown 與 browser kill
- [ ] 若採用 package-01 的 windowsgui 變體，此問題影響範圍會縮小，需在文件中標明兩者的關聯
- [ ] 文件化 Windows 平台下關閉程式的實際收尾行為

## 相關
- 檔案: cmd/nrf-factory/main.go
- 依賴: package-01
