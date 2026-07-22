# release-01: 版號策略（SemVer + tag + CHANGELOG）

- 階段: Release
- 優先: P1
- 狀態: todo

## 背景
目前尚未定義任何版號規範或 git tag 慣例。工具鏈版本（nrfutil、J-Link）已經鎖定，但應用程式本身還沒有版號機制可循。

## 驗收條件
- [ ] 採用 SemVer，並定義 git tag 慣例為 `vMAJOR.MINOR.PATCH`
- [ ] 定義何時該 bump 版號，例如 UI 或行為變更、燒錄參數變更等情境
- [ ] 新增並維護 CHANGELOG.md
- [ ] 與 build-02 的版本注入機制串接，讓 tag 能直接對應到二進位檔內顯示的版本

## 相關
- 檔案: CHANGELOG.md（待建）
- 依賴: build-02, release-02
