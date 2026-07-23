# package-04: Windows 程式碼簽章 / SmartScreen

- 階段: Package
- 優先: P2
- 狀態: todo

## 背景
未簽章的 exe 若是從網路下載或透過分享方式複製，會帶有 Mark-of-the-Web 標記。依 MOTW 機制預期會被 Windows SmartScreen 攔截；VM 驗證流程是以 `Unblock-File` 先行去除標記繞過（見 .docs/reviews/windows-parallels-verification.md，未實測攔截彈窗本身），仍需「解除封鎖」或 Run anyway 類操作才能直接執行。

## 驗收條件
- [ ] 評估以下兩個方向：(a) 購置 code signing 憑證為 exe 簽章以消除 SmartScreen 攔截、(b) 不簽章並在文件中清楚說明解除封鎖操作（檔案內容右鍵 → 解除封鎖，或使用 `Unblock-File`）
- [ ] 決策並記錄對應的成本與導入流程
- [ ] 更新 README 的產線說明，內容需對齊 VM 實測結果

## 相關
- 檔案: README.md
- 依賴: package-02
