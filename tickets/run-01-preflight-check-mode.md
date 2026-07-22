# run-01: 站台 preflight -check 模式

- 階段: Run
- 優先: P2
- 狀態: todo

## 背景
目前 preflight 檢查只會在開啟 UI 時執行。站台在 bring-up 階段常常想直接用命令列一次性核對環境是否就緒，不需要每次都開啟完整介面。

## 驗收條件
- [ ] 新增 `-check` 旗標，執行時不開啟服務或瀏覽器
- [ ] 印出 nrfutil core / device 版本、偵測到的 J-Link，以及 `nrfutil device --version` 回報的 tested 版本（V9.24a 來源）
- [ ] 各檢查項目分別顯示 pass/fail
- [ ] 只要有任一項缺失，程式以非零 exit code 結束
- [ ] `-check` 模式沿用與 UI 啟動時相同的 preflight 邏輯，不重複實作

## 相關
- 檔案: cmd/nrf-factory/main.go, docs/INSTALL-AND-RUN.md
- 依賴: -
