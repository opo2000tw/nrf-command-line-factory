# nRF Factory

工廠端 nRF 韌體燒錄桌面程式。啟動後在本機 `127.0.0.1` 開單一頁面，選外部 `.hex`、選 L/R、燒錄，並以彩色 log 與成功計數回報結果。

主目標平台：**Windows 10 / 11 x64**。macOS（Apple Silicon）可作開發與驗證。

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

| 項目 | 版本 | 用途 |
|------|------|------|
| [nRF Util](https://www.nordicsemi.com/Products/Development-tools/nRF-Util) + `nrfutil install device` | core **8.2.0** · device **2.19.1** | 列舉與燒錄 |
| [SEGGER J-Link Software](https://www.segger.com/downloads/jlink/) | **依 nrfutil 報的 tested 版**（device 2.19.1 對應 **V9.24a**；取法見下節，勿以某台機器現裝版為準） | probe 驅動；`JLinkExe` / `JLink` 在 `PATH` |
| USB **剛好一顆** J-Link | — | 0 顆或多顆都會在燒錄時報錯 |

可選：`NRFUTIL_PATH` 指向 `nrfutil` 完整路徑。

```bash
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
```

### nRF Util 元件與版本

nrfutil 是分層的：core 只是 launcher，實際燒錄由 `device` 子命令做，J-Link 又是另一包 SEGGER 軟體。三者版本各自獨立。

| 元件 | 現行版本 | 查版指令 | 角色 |
|------|----------|----------|------|
| nrfutil core | 8.2.0 | `nrfutil --version` | launcher；只負責裝／升子命令，不碰 probe |
| nrfutil device | 2.19.1 | `nrfutil device --version` | 載入 J-Link DLL、真正燒錄 |
| J-Link（tested） | V9.24a | 見下方官方取值 | device 命令**被測試對應**的 SEGGER 版 |

**J-Link 該裝哪版：以 nrfutil 官方輸出報的 tested 版為準，不是某台機器剛好裝的版本。** 這個版本綁在 device 命令上，升 device 後可能改變：

```bash
# 人類可讀：印 Detected（本機現裝）與 tested（對應）兩個版本
nrfutil device --version

# 只取「對應的 J-Link 版本」（機器無關、可腳本化）
nrfutil device --version --json \
  | jq -r '.. | objects | select(.name=="JlinkARM") | .expectedVersion.version'
# → JLink_V9.24a   （對應 device 2.19.1）
```

nrfutil 會註明 tested 版**非強制**：裝較新的 J-Link 通常照跑；只有遇到 J-Link 相關異常，才建議切回 tested 版（現為 V9.24a）。

**版本管理指令**

| 指令 | 作用 |
|------|------|
| `nrfutil list` | 列出已安裝子命令與版本 |
| `nrfutil search` | 列出可安裝子命令 |
| `nrfutil install device` | 安裝 device；可指定版本 `nrfutil install device=2.19.1`（`name[=version]`） |
| `nrfutil upgrade` / `nrfutil upgrade device` | 升級全部／指定子命令到最新 |
| `nrfutil self-upgrade` | 升級 core；`--to-version 8.2.0` 可指定或降回特定 core 版 |

### 取得執行檔

`dist/` 不進版控（已 gitignore）。權威建置方式是 `Makefile`（在已安裝 Go 的機器上）：

```bash
make dist       # 交叉編譯 windows/amd64、darwin/arm64 到 dist/
make package    # 承上，另外把各平台打包成 zip 並寫出 dist/SHA256SUMS
```

| 檔名 | 平台 |
|------|------|
| `nrf-factory-windows-amd64.exe` | Windows 10/11 x64（主目標） |
| `nrf-factory-darwin-arm64` | macOS Apple Silicon |

`make package` 會為每個平台產出 `dist/nrf-factory-<version>-<os>-<arch>.zip`（內含執行檔、README、docs/INSTALL-TOOLS.md，mac 平台再加 `exec.command`），以及對應的 `dist/SHA256SUMS`。

建置會用 `-ldflags -X` 把版本注入執行檔（`main.version` / `main.commit` / `main.date`），版本字串取自 `git describe --tags --always --dirty`；執行 `<binary> -version` 會印出目前版本。

若不透過 Makefile、手動 `go build` 且未帶上述 `-ldflags`，版本會 fallback 顯示為 `dev`：

```bash
# 手動單平台建置範例（fallback，非權威方式；正式建置請用 make dist / make package）
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
  -o dist/nrf-factory-windows-amd64.exe ./cmd/nrf-factory
```

推送 `v*` git tag 會觸發 GitHub Actions 的 Release workflow：自動建置、`make package`，並以 `gh release create` 附上各平台 zip 與 `SHA256SUMS`。版本規則與逐版變更見 [CHANGELOG.md](CHANGELOG.md)。

### 啟動

執行檔為單一自足程式：啟動後綁定本機 loopback，並自動以獨立 profile 的 Chrome / Edge app 視窗開啟畫面。**關閉該 app 視窗，或在主控台按 Ctrl+C，都會停止服務**（獨立 profile，關窗即結束該 browser process，程式接著自動 graceful shutdown）。

雙擊啟動：

| 平台 | 動作 |
|------|------|
| Windows | 進 `dist/` 資料夾，檔案總管雙擊 `nrf-factory-windows-amd64.exe` |
| macOS | Finder 雙擊 `exec.command`（開終端機並跑 `dist/` 內執行檔） |

> release zip 解壓後的結構與 `make dist` 完全一致：執行檔在 `dist/`，`exec.command` 與文件在根層。因此本節所有 `dist/…` 路徑在「解壓的 release zip」與「從原始碼建置」兩種情境都適用。

或在終端直接跑：

```bash
./dist/nrf-factory-darwin-arm64          # macOS Apple Silicon
```

啟動行為：

1. 綁定 `127.0.0.1` 隨機 port（不對外網開放），主控台印出 `nRF Factory: http://127.0.0.1:<port>`。
2. 找到 Chrome / Edge / Chromium / Brave 其一時，開獨立 app 視窗（專用 profile）；找不到時退回開系統預設瀏覽器分頁。
3. 關閉 app 視窗 → 服務自動停止；Ctrl+C 亦停止服務並關閉該 app 視窗。

| 旗標 / 變數 | 說明 |
|-------------|------|
| `-addr` | loopback 監聽位址，預設 `127.0.0.1:0`（隨機 port）；工廠可固定如 `127.0.0.1:17832` |
| `-no-browser` | 只起服務、不開瀏覽器；用主控台印的網址自行開啟，Ctrl+C 停止 |
| `-version` | 印出版本（`nRF Factory <version> (<commit>, built <date>)`）後結束程式，不開瀏覽器 |
| `NRFUTIL_PATH` | `nrfutil` 不在 `PATH` 時指向完整路徑 |
| `NRF_FACTORY_PROFILE` | 覆寫 app 視窗使用的 Chrome profile 目錄（預設在系統暫存區） |

固定 port 範例：

```bash
./dist/nrf-factory-darwin-arm64 -addr 127.0.0.1:17832
```

需已安裝 Chrome、Edge、Chromium 或 Brave 其一才會走 app 視窗；否則自動退回一般分頁。

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
| 關掉 app 視窗後服務就停了 | 這是預設行為（關窗＝停服務） | 想保留服務改用 `-no-browser`，再自行開網址 |

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

`make fmt` / `make vet` / `make test` 包裝上方對應指令；`make build` 建置本機平台執行檔到 `dist/`。交叉編譯與打包見上方「取得執行檔」。

原始碼入口：`cmd/nrf-factory/`（Go + 內嵌 `web/index.html`）。實作計畫見 `PLAN.md`。
