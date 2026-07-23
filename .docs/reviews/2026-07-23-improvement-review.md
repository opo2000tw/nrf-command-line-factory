# 三方改進審查:codex + sonnet + grok

日期:2026-07-23
對象:v0.1.3 之後的全倉盤點,涵蓋 Go 後端(cmd/nrf-factory/main.go、browser.go)、web UI(index.html)、Playwright 測試、CI/release workflow、Makefile 與文件/tickets。
方式:三方獨立審查後由主會話彙整。sonnet 全倉逐檔讀碼;codex 以 safety/correctness/operational lens 審查;grok 提供獨立第二意見與外部實務對照。主會話負責去重、對照原始碼核驗、排優先序與更正。sonnet 與 codex 皆實際讀碼;grok 依架構摘要推論,其中數點經核驗後下修(見末節)。

## 已於本次一併處理

- 前端 busy 競態(P1):flash 送出前重新上鎖,避免 pre-flight ping 把 UI 解鎖。
- 燒錄/安裝脫離 request context(P1):瀏覽器斷線或關窗不再 cancel 正在進行的 nrfutil;關窗時若 busy 會等待完成(可再次 Ctrl+C 強制),前端加 beforeunload 確認。
- CI/release 快贏(P2):CI Node 升版、消除 go.sum 快取警告;release 加 SHA256SUMS 回驗與 zip 內容驗證。
- 對應測試補齊,其中原本以假 state 繞過的 busy-lock 測試改為驗證真實鎖定行為。

其餘依嚴重度列於下,未即時處理者以 tickets 追蹤(test-02、security-01、release-03)。

## P1 高優先(安全可利用 / 硬體風險 / 治理)

| 項目 | 提出方 | 說明與方向 |
|------|--------|-----------|
| 前端 busy 鎖被 pre-flight ping 解除 | sonnet+codex | 已處理(見上)。原測試以假 state 掩蓋,已改為真實驗證。 |
| flash/install ctx 綁 request context | sonnet+codex+grok | 已處理。斷線/關窗會腰斬 erase/program,DUT 有半寫入風險。 |
| /api/tool 任意上傳檔 chmod 0755 直接執行 | 三方 | RCE 向量。方向:移除 web 上傳可執行檔隨包已驗證工具,或僅收簽章/allowlisted SHA-256。列入 security-01。 |
| 固定 /tmp 路徑 + O_TRUNC 的 symlink/TOCTOU | sonnet+codex+grok | 應改 app 私有 0700 隨機目錄 + O_EXCL + 非 symlink 驗證。列入 security-01。 |
| loopback 無 CSRF / DNS-rebinding 防護 | codex+grok | 無 Origin/Host/token 檢查。方向:per-launch token + 精確 Origin/Host 驗證。列入 security-01。 |
| 單一 J-Link 不等於正確 J-Link | codex+grok | 插錯 probe 會安全地燒到錯設備;list 與 program 間有裝置變更窗口。方向:serial allowlist 綁定。列入 security-01。 |
| README/PLAN 授權治理矛盾 | sonnet | PLAN.md 明文「nrfutil 各帶授權、不由本程式重散布」,v0.1.2 內附 nrfutil 推翻此原則卻無授權重評。列入 release-03。 |
| Windows 未簽章 / macOS 未 notarize | 三方 | 產線安裝受阻且無法驗證未被替換。既有 package-03/package-04 追蹤。 |

## P2 正確性 / 可追溯 / 可維運

| 項目 | 提出方 | 說明與方向 |
|------|--------|-----------|
| busy 為單一全域 bool,未含所有互斥操作 | codex+grok | 燒錄中可替換工具或重設計數。方向:operation manager + 明確狀態機。 |
| 計數不持久化 + 無稽核 log | 三方 | 產線報工與追溯失效。方向:結構化 log + append-only ledger(operation ID、firmware hash、probe SN、階段、結果)。 |
| release workflow 缺完整 gate | sonnet+codex | 已加 checksum 與內容驗證;仍待補 gofmt/vet/ui-test 對同一 SHA 的 gate 與 OS 相符的執行檔 smoke。列入 test-02/release 後續。 |
| make package 從未在 CI dry-run | sonnet+codex | 打包鏈只有正式 tag 才第一次執行。方向:PR 期跑一次 dry-run。 |
| 3rd/ 二進位未走 git-lfs + 無供應鏈 manifest | 三方 | 違反大檔用 git-lfs 慣例;缺來源/版本/SHA-256/授權 manifest。列入 release-03。 |
| http.Server timeout 不足 | codex | 已設 ReadHeaderTimeout;缺 IdleTimeout/MaxHeaderBytes(NDJSON 串流需 heartbeat 而非短 WriteTimeout)。 |
| CommandContext 不 kill process group | sonnet+codex | nrfutil 孫行程可能殘留(對應 browser.go 既有簡化註解)。2026-07-23 macOS 核驗補充:不只孫行程——force-stop 路徑(二次 Ctrl+C / 等待逾時)只讓 main 返回,燒錄中的 nrfutil 本身的 ctx 從未被 cancel,會變孤兒行程繼續寫韌體,下次啟動 J-Link 可能仍被佔用。 |
| newApp 未含 deviceReady,靠呼叫端補設 | 2026-07-23 macOS 核驗 | 目前唯一 production call site 有補,無現行 bug;但該行被移除時 deviceReady 會靜默退回 false,屬維護 foot-gun。 |
| installedJLinkVersion 兩次呼叫共用 5 秒 ctx | 2026-07-23 macOS 核驗 | 第一次呼叫吃掉大半預算時 fallback 會被時間壓垮;實測回應 <100ms 未觸發,防禦性強化項。 |
| 前端非 .hex 無 client 驗證 | 三方 | 選錯檔要一次網路來回才知。方向:change handler 加副檔名檢查。 |
| eventWriter 無界 + 進度回車 | sonnet+codex | nrfutil 若以回車印進度,事件與 DOM 可能膨脹;需實機輸出定嚴重度。 |
| clearSingletonLocks 不驗行程存活 | sonnet | 雙開會清掉仍在用的 Chrome lock,毀 profile。與 run-02 同源。 |
| 逾時常數寫死、不可注入 | sonnet+codex | 最危險的 cancel/timeout 分支無法回歸測試。方向:改成可覆寫欄位。 |
| busy 時前端停止 polling | codex | 其他分頁 poll 到 busy 後不再更新,永久卡住。 |
| install-device 不 pin 版本 + 需網路 | codex | 工廠離線/代理故障即癱瘓。方向:pin 版本 + offline plugin。 |
| 測試缺口 | 三方 | 多/零 J-Link、逾時、串流中途錯誤、真實 busy、Go 表格分支(409/405/缺檔/超限)、race detector、有限度 HIL。列入 test-02。 |

## P3 健壯性 / 體驗 / 維護

- handleEvent 的 JSON 解析無錯誤保護,串流中斷會顯示原始語法錯誤。
- 錯誤訊息把子行程原始輸出直接回前端(loopback 風險低,建議 log 詳情、UI 顯示精簡)。
- 無障礙:計數容器無 aria-live;無自動 a11y 掃描;產線手套情境的 tab order 與大按鈕。
- Chrome profile 目錄永不清理。
- CI 缺 npm 與 Playwright 瀏覽器快取(已補 npm 快取)。
- GitHub Actions 未 pin 到 commit SHA、權限最小化。
- 既有的兩處刻意簡化(kill 不含 process group、J-Link 只做存在檢查)只在行內註解,未進 ticket。
- 設定檔(站別/允許 serial/逾時)、健康檢查端點、visibilitychange 暫停 polling、i18n 與高對比。

## 對三方輸出的核驗更正

- grok「缺 body timeout / 上傳無大小上限」:已有 maxUploadBytes(64MB)/toolUploadBytes(128MB)與 MaxBytesReader,下修。
- grok「flash 無逾時,busy 永久占用」:已有 5 分鐘 flashTimeout,下修為 nuance。
- grok「應設 ReadHeaderTimeout」:已設(5 秒);缺的是其餘 timeout。
- codex「readiness 只靠一次 --version」:preflight 其實已查 nrfutil --version、device --version 與 J-Link 存在,部分已做。
- codex/grok「CSP 應加 connect-src」:已含 connect-src self;真正缺的是 frame-ancestors/object-src/base-uri。
- 版本一致性(CHANGELOG 對 tag):已核對一致(v0.1.3)。

## 追蹤

- test-01:Playwright UI 功能測試套件(done,記錄已完成範圍)。
- test-02:測試覆蓋缺口(todo)。
- security-01:loopback server 與工具上傳的安全強化(todo,P1)。
- release-03:nrfutil 內附散布的授權確認與供應鏈 manifest(todo,P1)。
