# build-01: 可重現的多平台建置腳本

- 階段: Build
- 優先: P1
- 狀態: done

## 背景
目前三平台的建置指令只寫在 README 裡，要求開發者手動輸入 `GOOS`/`GOARCH` 搭配 `go build -trimpath -ldflags "-s -w"`。這種手抄流程沒有單一來源，容易在不同人手上、不同時間點漂移，導致三平台實際建置出來的旗標或輸出路徑不一致。

## 驗收條件
- [x] 有單一入口（Makefile 的 `make dist` 目標）可一次產出 windows/amd64、darwin/arm64 兩個平台的執行檔到 `dist/`
- [x] 輸出檔名對齊 README 目前描述的命名方式
- [x] 建置維持純 Go、不使用 cgo，並沿用 `-trimpath -ldflags "-s -w"`
- [x] 腳本可透過參數只建置單一平台，方便本機除錯
- [x] README 的建置章節改為指向此腳本，不再重複列出手動指令

## 相關
- 檔案: cmd/nrf-factory, README.md
- 依賴: build-02
