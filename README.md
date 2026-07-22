# nRF Factory

工廠端 nRF 韌體燒錄桌面程式。啟動後在本機 `127.0.0.1` 開單一頁面，選外部 `.hex`、選 L/R、燒錄，並以彩色 log 與成功計數回報結果。

主目標平台：**Windows 10 / 11 x64**。macOS（Apple Silicon / Intel）可作開發與驗證。

---

## 需求摘要

原始需求（PC Win 10/11 工廠燒錄）：

Vensan Lin · topteam technology taiwan co., ltd  
tel. 886-4-22312859 · fax. 886-4-22362118

1. 燒錄檔案不嵌入程式，執行時由操作者選取。
2. 系統：Windows 10、Windows 11。
3. 佈局：
   - 顯示選取的檔名
   - L / R 側別按鈕（亮暗）
   - 燒錄、清除計數
   - Terminal：初始／燒錄／清除為白，失敗紅，成功綠
   - L、R 成功計數
4. 工具環境：
   - 啟動時偵測 nRF Util（`nrfutil device`）與 SEGGER J-Link
   - 缺少時在畫面列出缺哪一項與安裝方式
   - 工具不嵌入程式，由工站自行安裝

---

## 使用說明

### 工站前置條件

程式本身**不附帶** Nordic / SEGGER 工具，工站需先裝好。  
**固定版本與官方下載步驟**見 **[docs/INSTALL-TOOLS.md](docs/INSTALL-TOOLS.md)**（對齊本開發機已驗證組合）。

| 項目 | 本機已驗證版本 | 用途 |
|------|----------------|------|
| [nRF Util](https://www.nordicsemi.com/Products/Development-tools/nRF-Util) + `nrfutil install device` | core **8.2.0** · device **2.19.1** | 列舉與燒錄 |
| [SEGGER J-Link Software](https://www.segger.com/downloads/jlink/) | **V9.60** | probe 驅動；`JLinkExe` / `JLink` 在 `PATH` |
| USB **剛好一顆** J-Link | — | 0 顆或多顆都會在燒錄時報錯 |

可選：`NRFUTIL_PATH` 指向 `nrfutil` 完整路徑。

```bash
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
```

### 取得執行檔

`dist/` 不進版控（已 gitignore）。由本機交叉編譯產出，例如：

| 檔名 | 平台 |
|------|------|
| `nrf-factory-windows-amd64.exe` | Windows 10/11 x64（主目標） |
| `nrf-factory-darwin-arm64` | macOS Apple Silicon |
| `nrf-factory-darwin-amd64` | macOS Intel |

自行建置（在已安裝 Go 的機器上）：

```bash
# Windows x64（可在 macOS / Linux 交叉編譯，純 Go 無 cgo）
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o dist/nrf-factory-windows-amd64.exe ./cmd/nrf-factory

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" \
  -o dist/nrf-factory-darwin-arm64 ./cmd/nrf-factory

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o dist/nrf-factory-darwin-amd64 ./cmd/nrf-factory
```

### 啟動

#### 建議：app 視窗（Chrome / Edge `--app=`）

不開一般瀏覽器分頁，改用獨立 app 視窗。預設固定 **`http://127.0.0.1:17832`**。

**生命週期：關掉 app 視窗 = 停止 nRF Factory 服務**（獨立 Chrome profile，關最後一個窗就結束該 process，腳本接著 kill server）。

```bash
./exec.sh              # 起服務 + 開窗，擋在終端；關窗後自動 stop
./exec.sh open         # 只重開窗（服務須已在跑；不接管生命週期）
./exec.sh status
./exec.sh stop         # 強制停服務（並試著關 app Chrome）
```

**雙擊啟動**

| 檔案 | 平台 | 說明 |
|------|------|------|
| `exec.command` | macOS | Finder **雙擊**會開「終端機」並跑 `exec.sh`（`.sh` 本身雙擊通常不會執行） |
| `exec.bat` | Windows | 檔案總管雙擊；有 Git Bash 時走完整 `exec.sh` session，否則直接開 `.exe` |

第一次若 macOS 擋 `exec.command`：右鍵 → 打開，或：

```bash
chmod +x exec.command exec.sh scripts/browser-app.sh
xattr -d com.apple.quarantine exec.command   # 若從下載/AirDrop 來
```

| 指令 / 變數 | 說明 |
|-------------|------|
| `./exec.sh` / `start` | session：起服務 → 開窗 → **關窗即停服務** |
| `./exec.sh open` | 僅再開一個 app 窗 |
| `./exec.sh stop` | 強制停止 |
| `./exec.sh status` | pid / url |
| `NRF_FACTORY_BIN` | 指定執行檔 |
| `NRF_FACTORY_ADDR` | 預設 `127.0.0.1:17832` |
| `.run/` | pid / url / log / chrome-profile（gitignore） |

```bash
NRF_FACTORY_BIN=./dist/nrf-factory-darwin-arm64 ./exec.sh
```

需已安裝 Chrome、Edge、Chromium 或 Brave 其一。

#### 直接跑執行檔

**Windows**

```text
nrf-factory-windows-amd64.exe
```

**macOS**

```bash
./dist/nrf-factory-darwin-arm64          # Apple Silicon
# 或
./dist/nrf-factory-darwin-amd64          # Intel
```

行為：

1. 綁定 `127.0.0.1` 隨機 port（不對外網開放）。
2. 主控台印出 `nRF Factory: http://127.0.0.1:<port>`。
3. 預設自動開系統瀏覽器（一般分頁）；不要開瀏覽器時加 `-no-browser`。

```bash
./dist/nrf-factory-darwin-arm64 -no-browser
# 另開 app 視窗：
./scripts/browser-app.sh http://127.0.0.1:PORT
```

關閉服務：用 `./exec.sh` 時 **關掉 app 視窗** 即停；或 `./exec.sh stop` / 終端 `Ctrl+C`。直接前景跑 binary 時，在該終端 `Ctrl+C`。

#### macOS Gatekeeper

本機 `go build` 的產物通常沒有 quarantine，可直接跑。若從下載／AirDrop 複製到別台 Mac，可能被擋：

```bash
xattr -d com.apple.quarantine nrf-factory-darwin-arm64
```

或在 Finder 右鍵 → 開啟。執行檔未簽章、未 notarize。

### 畫面操作（正常燒錄）

1. 確認右上狀態點為**綠**，並顯示類似：
   `nrfutil … · nrfutil-device … · J-Link OK`
   - 紅點：缺 nrfutil、缺 `device` command、或缺 J-Link；依訊息安裝後**重開程式**。
2. 點選 **.hex** 韌體（僅支援 `.hex`；檔案只在本次燒錄暫存，結束後刪除，不寫進程式）。
3. 選 **L** 或 **R**（產品側別與成功計數分類；第一版**不**對應不同 probe）。
4. 確認「開始燒錄 L/R」可按後點下。
5. 右側 terminal 串流 nrfutil 輸出：
   - 一般過程：白
   - 成功：綠（例如「L 側燒錄成功」），該側計數 +1
   - 失敗：紅，計數**不**增加
6. **清除計數**：L/R 成功次數同時歸零（只存在記憶體，重開程式也會歸零）。

燒錄中會鎖定檔案選擇、側別與清除；不可並行第二筆。

### 燒錄時程式實際做的事

1. `nrfutil device list --traits jlink --json` — 必須剛好一顆 J-Link。
2. `nrfutil device program`，選項固定為：
   - `chip_erase_mode=ERASE_RANGES_TOUCHED_BY_FIRMWARE`（只清韌體觸及範圍，非整片 mass erase）
   - `verify=VERIFY_READ`
   - `reset=RESET_SYSTEM`
3. 逾時上限 5 分鐘；上傳上限 64 MiB。

**注意**：燒錄會改寫目標 flash 並 reset 晶片。接上真機前請確認 `.hex` 與側別（L/R）正確。

### 建議驗收順序

| 步驟 | 是否改寫目標 | 說明 |
|------|--------------|------|
| 啟動後右上綠燈 | 否 | preflight 通過 |
| 只接一顆 J-Link | 否 | 可用 `nrfutil device list --traits jlink` 對照 |
| 選 L 側 `.hex` → 燒錄 | **是** | log 綠字成功、L 計數 +1、目標應重開 |
| 選 R 側 `.hex` → 燒錄 | **是** | 同上，R 計數 +1 |
| 拔掉 probe 再燒 | 否（應失敗） | 紅字、計數不變 |
| 清除計數 | 否 | L/R 皆 0 |

倉庫 `test/` 可放現場用的 L/R hex（若有）；勿把專有韌體 commit 進版控。

### 常見問題

| 現象 | 可能原因 | 處理 |
|------|----------|------|
| 狀態紅：找不到 nrfutil | 未安裝或未進 PATH | 安裝 nRF Util；或設 `NRFUTIL_PATH` |
| 狀態紅：缺少 device command | 只裝了 core | `nrfutil install device` |
| 狀態紅：找不到 J-Link | 未裝 J-Link Software | 安裝 SEGGER 套件並確認 PATH |
| 燒錄失敗：找不到 J-Link | USB 未接或驅動異常 | 重插、檢查 `device list` |
| 燒錄失敗：找到多個 J-Link | 同時接了兩顆以上 | 只留一顆再燒 |
| 開始燒錄按鈕灰的 | 尚未選 `.hex`，或正在燒錄 | 先選檔；等 busy 結束 |
| 非 `.hex` 被拒 | 第一版只收 hex | 換成 `.hex` |
| macOS 打不開 | Gatekeeper quarantine | 見上方 `xattr` / 右鍵開啟 |
| 關掉瀏覽器後畫面沒了 | 服務仍在跑 | 用主控台印的 URL 再開，或重啟程式 |

### 安全與範圍（第一版）

- 只聽 loopback，不開對外 port。
- 不提供 recover / 獨立 mass erase。
- 不嵌入韌體、nrfutil、J-Link。
- 同時只允許一個燒錄工作。
- L/R 是產品側別與計數，不是雙 probe 對應；若治具改雙 probe，需另做 serial 對應。

---

## 開發指令

```bash
go mod tidy
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

原始碼入口：`cmd/nrf-factory/`（Go + 內嵌 `web/index.html`）。實作計畫見 `PLAN.md`。
