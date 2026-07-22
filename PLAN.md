# nRF Factory 實作計畫

## 目標與範圍

建立可在 macOS 與 Windows 執行的 Go 程式。啟動後只在 `127.0.0.1` 開啟本機服務並顯示單頁 SPA，提供外部韌體選擇、L/R 選擇、燒錄、彩色 log、L/R 成功計數與清除功能。韌體不得編入執行檔，計數先只保存在記憶體。

## 最小技術方案

- GUI 不使用 Wails、Fyne、React 或 Node。Go 標準庫 `net/http` 提供 API，Vanilla HTML/CSS/JavaScript 負責單頁介面；`//go:embed` 只收錄 UI 資產，nRF Util 與 J-Link 不嵌入。
- 瀏覽器以系統原生方式開啟。服務使用 loopback 自動分配的 port，不接受外部網路連線。
- UI 以上傳方式送入外部 `.hex` 韌體；後端寫入權限受限的暫存檔，燒錄結束後刪除。
- 燒錄後端使用 Nordic 現行的 `nrfutil device`，由 `os/exec.CommandContext` 直接傳入參數，不經 shell。合併 stdout/stderr 後串流到畫面。不得改用已淘汰的 `nrfjprog`。
- `nrfutil`、`device` command 與 `SEGGER J-Link` 皆不嵌入（各帶 Nordic／SEGGER 授權，不由本程式重散布）；改為啟動時 preflight 偵測是否安裝，缺哪一項就在畫面明確顯示缺什麼與安裝指引。firmware 同樣不嵌入，runtime 由使用者外部選取。
- 第一版只支援 J-Link 連接與 `.hex`。若偵測不到相容裝置或同時存在多個裝置，直接顯示錯誤，不先做裝置設定頁。
- L/R 暫定為產品側別與計數分類，不映射不同 probe。若實際治具是雙 probe，再加入 serial number 對應。
- TinyGo 不採用：它的 `flash` 流程面向「編譯指定 board target 的 Go 原始碼」，不是工廠端任意外部韌體燒錄器。
- 不提供 `recover` 或獨立 mass erase。正式實機測試前，依目標晶片確認 erase、verify 與 reset 選項，且使用已安裝版本的 `nrfutil device program --help` 作為準據。

預計只建立 `go.mod`、`cmd/nrf-factory/main.go`、同目錄測試檔與 `cmd/nrf-factory/web/index.html`。先不建立 interface、資料庫、設定框架或多 package 分層。

## 平台產物與環境需求

- macOS 與 Windows 共用同一份 Go/SPA 原始碼，但執行檔必須分平台編譯，不能互相執行。
- 產物是單一 Go 執行檔，只內嵌 UI；不夾帶任何 `nrfutil`、`device` command 或 J-Link 資產，因此不需平台專屬的嵌入 build-tag 檔。
- 工站須自行安裝 nRF Util（並執行 `nrfutil install device`）與 SEGGER J-Link；這些工具各帶 Nordic／SEGGER 授權，不由本程式重散布。nRF Command Line Tools 已封存，改用 nRF Util。
- `nrfutil` 由 `PATH` 或 `NRFUTIL_PATH` 探測。啟動 preflight 逐項檢查並回報：找不到 `nrfutil`、缺 `device` command、或 J-Link 不可用時，分別在畫面顯示「缺哪一項」與對應安裝指引；缺工具不阻擋 UI 載入，燒錄仍可觸發，但會在 `nrfutil device list` 裝置列舉階段以明確錯誤中止，不誤動作或誤增計數。
- 版本不由程式固定；preflight 讀出實際安裝的 `nrfutil` 與 `device` 版本顯示在狀態列，方便現場核對與更新後重測。

## 逐步實作與測試

### 階段一：可啟動的 SPA

- 建立 Go module、本機 HTTP server、SPA 靜態頁與自動開啟瀏覽器。
- 完成檔案選擇、檔名顯示、L/R 亮暗狀態、燒錄按鈕、清除按鈕、log 區與計數顯示。
- 以 `go fmt ./...`、`go test ./...`、`go vet ./...`、`go build ./...` 為 gate。
- 手動確認頁面只能由 loopback 存取，重新整理後仍能讀取目前記憶體狀態。

### 階段二：無硬體的燒錄流程

- 實作上傳驗證、暫存檔生命週期、同時間只允許一個燒錄工作，以及取消與逾時處理。
- 將 command runner 注入測試，用假程序模擬成功、非零 exit code、找不到命令、逾時與連續 log。
- 驗證只有成功才增加所選側計數；失敗不得增加；清除同時歸零 L/R。
- 驗證檔名與參數不會進入 shell，錯誤內容會以紅色顯示，成功以綠色顯示，其餘 log 使用一般色。

### 階段三：nRF Util 唯讀整合

- 啟動 preflight 分項檢查 `nrfutil --version`、`nrfutil device --version` 與 SEGGER J-Link 可用性；任一缺少就在 GUI 指出「缺哪一項」與安裝方式；工具缺失時實際燒錄會在裝置列舉階段以錯誤中止。
- 使用 `nrfutil device list` 找出支援的 J-Link serial number；如需機器解析，使用官方 JSON Lines 輸出而非解析人類可讀文字。
- 在 macOS 與 Windows 分別跑版本檢查、裝置列舉及錯誤情境；本階段不執行 erase 或 program。

### 階段四：macOS 實機燒錄

- 接上已知型號的 nRF 目標與 J-Link，先以命令列驗證同一份 `.hex` 和選定的 erase/verify/reset 行為。
- 從 SPA 依序測試 L 成功、R 成功、拔除 probe、錯誤韌體與清除計數。
- 確認燒錄完成後目標會啟動、成功計數正確、失敗不計數、暫存韌體已移除。

### 階段五：Windows 建置與實機驗收

- 在原生 Windows runner 執行格式檢查、單元測試、vet 與 build，產出 `.exe`；macOS 也在原生 runner 產出對應執行檔。
- 在 Windows 10 或 11 安裝 Nordic/SEGGER 必要元件，重跑與 macOS 相同的唯讀及實機案例。
- 最後驗證路徑含空白與非 ASCII 字元、沒有 `nrfutil`、沒有裝置、燒錄中重複點擊及程式關閉等情境。

## 實機前必須確認

- 目標 nRF 型號、板型與連接方式是 J-Link、Nordic DFU 或其他 transport。
- L/R 是產品分類、韌體變體，還是兩個實體燒錄 probe。
- 允許的韌體格式，以及是否可全片 erase、是否必須保留 UICR 或其他出廠資料。
- 燒錄後要 reset、斷電重上電，或停留在 programming 狀態。

這些答案不阻擋 SPA、狀態管理與假程序測試，但在第一次實機 program 前必須固定。

## 完成條件

- macOS 與 Windows 原生建置及啟動驗證通過。
- README 所列 UI 行為全部可操作，外部韌體未嵌入程式。
- 缺少工具、缺少裝置及燒錄失敗均有清楚訊息，且不誤增成功計數。
- 同一份已知良好韌體在兩個作業系統完成實機燒錄與啟動驗證。

## 選型依據

nRF Command Line Tools 已封存，Nordic 建議改用本工具採用的 nRF Util，且該套件另含 SEGGER J-Link，屬工站自行安裝的前置需求，程式只負責偵測與回報：[nRF Command Line Tools](https://www.nordicsemi.com/Products/Development-tools/nRF-Command-Line-Tools)。

Nordic 官方文件說明 `nrfutil` 支援 Windows、Linux 與 macOS，`nrfutil device` 提供裝置列舉、program、verify、reset 等操作：[nRF Util prerequisites](https://docs.nordicsemi.com/r/bundle/nrfutil/page/guides/installing.html/prerequisites)、[device programming](https://docs.nordicsemi.com/r/bundle/nrfutil/page/nrfutil-device/guides/programming.html)。TinyGo 的官方範例則以 board target 編譯並燒錄 Go 程式，因此不作為本工具的主後端：[TinyGo nRF52840 flashing](https://tinygo.org/docs/reference/microcontrollers/boards/feather-nrf52840/)。
