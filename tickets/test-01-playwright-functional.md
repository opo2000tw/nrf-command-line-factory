# test-01: Playwright UI 功能測試套件

- 階段: Test
- 優先: P1
- 狀態: done

## 背景
web UI 的按鈕與輸入原本沒有自動化功能測試,回歸只靠手動。改以 Playwright 對真實內嵌頁面做 route-mock 測試,不需 nrfutil 或 J-Link 硬體即可涵蓋成功與失敗路徑。

## 驗收條件
- [x] webServer 以 `-no-browser` 固定埠啟動真實 Go app 服務內嵌頁面
- [x] `page.route` 攔截 /api/* 使 firmware/tool 選檔、L/R 選側、flash、reset、install 的成功與失敗路徑可決定性重現
- [x] firmware input:檔名/提示更新、KB 進位、選側順序無關、啟用邏輯
- [x] L/R 站台:active 與 aria-pressed 切換、鈕文字、計數不受點選影響
- [x] flash:L/R 成功計數、HTTP 錯誤、串流錯誤、離線、燒錄中鎖定
- [x] reset:清除成功、409 衝突、忙碌與離線停用
- [x] tool 上傳與 install-device:就緒/仍缺、HTTP 錯誤、忙碌鎖定、內附 3rd/ 命令顯示
- [x] CI 新增 ui-test job 一併執行
- [x] 後續擴充（v0.1.5）：偵測裝置按鈕（detect.spec.ts：J-Link/MCU 兩段結果、多裝置、忙碌鎖定、離線停用）、install 按鈕依 deviceReady 灰化、preflight 的 SEGGER J-Link 版本字串顯示

## 相關
- 檔案: tests/*.spec.ts, tests/helpers.ts, playwright.config.ts, .github/workflows/ci.yml
- 依賴: -
- 後續: test-02（缺口）
