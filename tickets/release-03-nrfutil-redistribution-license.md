# release-03: nrfutil 內附散布的授權確認與供應鏈 manifest

- 階段: Release
- 優先: P1
- 狀態: todo

## 背景
源自三方審查（見 .docs/reviews/2026-07-23-improvement-review.md）。PLAN.md 最初明文「nrfutil、device command 與 SEGGER J-Link 皆不嵌入(各帶 Nordic／SEGGER 授權,不由本程式重散布)」。v0.1.2 的變更「Bundle nrfutil in 3rd/ and auto-use it」把 nrfutil 內附進 release zip 並自動採用,實質推翻了這條以授權為由的設計原則,但沒有任何 ticket 記錄授權重新評估,PLAN.md 相關段落也未更新。此外 3rd/ 的二進位以一般 blob 提交,缺來源/版本/雜湊/授權的可稽核 manifest。

## 驗收條件
- [ ] 確認 nRF Util 的授權條款是否允許以此方式重新散布;結論寫入 .docs/decisions/
- [ ] 依結論更新 PLAN.md 相關段落(維持一致,不留與現況矛盾的敘述)
- [ ] release zip 附上 nrfutil 的 license / NOTICE
- [ ] 為 3rd/ 二進位加入 machine-readable manifest(upstream 來源、版本、SHA-256、簽章驗證方式、license),CI 比對實際檔案 digest
- [ ] 評估 3rd/ 二進位改用 git-lfs 管理(僅解儲存,不提供 authenticity)

## 相關
- 檔案: PLAN.md, README.md, Makefile, 3rd/
- 依賴: -
