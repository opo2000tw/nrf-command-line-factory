# package-03: macOS Gatekeeper / 簽章策略

- 階段: Package
- 優先: P3
- 狀態: todo

## 背景
目前的 macOS 執行檔未簽章、未 notarize。跨機複製時會被 quarantine 屬性擋下，目前只能靠 `xattr` 指令或右鍵開啟繞過，這個做法已寫在 README 裡，但仍是暫時性的權宜方案。

## 驗收條件
- [ ] 評估並決策以下三個方向之一：(a) 維持未簽章並在文件中說明 xattr 操作、(b) 採用 ad-hoc codesign、(c) 採用 Developer ID 簽章並送 notarize
- [ ] 若採用簽章方案，記錄所用憑證與 codesign / notarytool 的操作步驟
- [ ] 更新 README 與 docs/INSTALL-TOOLS.md 反映最終決策

## 相關
- 檔案: README.md, docs/INSTALL-TOOLS.md
- 依賴: package-02
