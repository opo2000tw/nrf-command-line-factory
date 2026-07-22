# build-02: 版本戳記嵌入執行檔

- 階段: Build
- 優先: P1
- 狀態: done

## 背景
目前產出的二進位檔沒有任何版本資訊，現場人員拿到一個 exe 時無法核對手上跑的到底是哪一版建置，出問題時也難以回報對應版本。

## 驗收條件
- [x] 以 `-ldflags "-X main.version=... -X main.commit=... -X main.date=..."` 在建置時注入版本資訊
- [x] 版本字串取自 `git describe --tags --always --dirty`
- [x] 新增 `-version` 旗標，執行後印出版本並直接結束程式
- [x] console 啟動 banner 與 UI footer 都顯示目前版本
- [x] 未注入版本資訊時（例如本機 `go build` 未帶 ldflags）要 fallback 顯示為 `dev`

## 相關
- 檔案: cmd/nrf-factory/main.go, cmd/nrf-factory/web/index.html
- 依賴: -（被 build-01、release-02 使用）
