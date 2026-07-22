# run-04: Windows 關 console 的 graceful shutdown

- 階段: Run
- 優先: P3
- 狀態: todo

## 背景
目前的 signal 處理只接了 `os.Interrupt`（Ctrl+C）。在 Windows 上關閉 console 視窗會觸發 `CTRL_CLOSE_EVENT`，目前的收尾行為只是 best-effort，可能來不及完成 server shutdown 或 kill 瀏覽器程序。

## 驗收條件
- [ ] 確認並盡量處理 Windows console close 事件下的 graceful shutdown，涵蓋 server shutdown 與 browser kill
- [ ] 若採用 package-01 的 windowsgui 變體，此問題影響範圍會縮小，需在文件中標明兩者的關聯
- [ ] 文件化 Windows 平台下關閉程式的實際收尾行為

## 相關
- 檔案: cmd/nrf-factory/main.go
- 依賴: package-01
