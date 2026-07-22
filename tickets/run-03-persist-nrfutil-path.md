# run-03: nrfutil 路徑持久化

- 階段: Run
- 優先: P3
- 狀態: todo

## 背景
新增的 in-app 選擇 nrfutil 路徑功能（`/api/tool`）目前只存在記憶體中，程式重開就會失效，導致站台每次都要重新選擇一次路徑。

## 驗收條件
- [ ] 將使用者選定的 nrfutil 路徑持久化，存放在 `os.UserConfigDir` 下的 nrf-factory 設定檔中
- [ ] 啟動時若設定檔內有效路徑存在，直接採用並執行 preflight；路徑無效則回退到預設路徑
- [ ] 定義持久化路徑與 `NRFUTIL_PATH` 環境變數之間的優先順序

## 相關
- 檔案: cmd/nrf-factory/main.go
- 依賴: -
