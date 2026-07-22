# nRF Factory 安裝與執行指南

本文件是**單一自足**的操作手冊：只靠這一份文件，工站操作者即可完成 toolchain 安裝與日常燒錄操作。分兩部分：

- **A. 安裝 toolchain** — nRF Util（core + device）與 SEGGER J-Link。
- **B. 執行** — 解壓 release zip 後如何啟動、操作畫面、常見問題。

nRF Factory 執行檔本身**不內嵌** Nordic / SEGGER 工具（授權限制），工站須依本文件自行安裝。

---

## A. 安裝 toolchain

### A.1 已驗證版本

下列版本為本專案開發機（macOS arm64）實測可用組合，作為工站安裝對齊基準：

| 元件 | 版本 | 查版指令 |
|------|------|----------|
| nRF Util（core） | **8.2.0** | `nrfutil --version` |
| nRF Util **device** command | **2.19.1** | `nrfutil device --version` |
| SEGGER J-Link Software | tested **V9.24a**（對應 device 2.19.1） | 見下方官方指令 |

**J-Link 該裝哪一版，一律以 nrfutil 官方輸出報的 tested 版為準，不是以某台機器現裝版為準。** 這個對應關係綁在 device 命令版本上，device 升版後可能改變。用下列機器無關、可腳本化的方式取得：

```bash
# 人類可讀：同時印出本機現裝與 tested 兩個版本
nrfutil device --version

# 只取「對應的 J-Link 版本」（機器無關，適合寫進安裝腳本）
nrfutil device --version --json \
  | jq -r '.. | objects | select(.name=="JlinkARM") | .expectedVersion.version'
# 目前 device 2.19.1 對應 → V9.24a
```

tested 版**非強制**：裝較新版的 J-Link 通常照跑（本機以 V9.60 實測 list / program 皆正常）；只有出現 J-Link 相關異常時，才建議切回 tested 版。

啟動後右上狀態列成功時會顯示類似字串：

```text
nrfutil 8.2.0 … · nrfutil-device 2.19.1 … · J-Link OK
```

### A.2 安裝 nRF Util

1. 官方下載頁：[nRF Util](https://www.nordicsemi.com/Products/Development-tools/nRF-Util)，安裝說明：[Installing nRF Util](https://docs.nordicsemi.com/bundle/nrfutil/page/guides/installing.html)。
2. 依作業系統下載官方安裝包（Windows / macOS / Linux）。**產線 Windows 請走 Nordic 官網安裝包**；Homebrew cask 只是 mac 開發機的便利路徑，不要在產線依賴 brew。
3. 安裝 core 之後，**必須**再安裝 device 子命令：

```bash
nrfutil install device
nrfutil list
nrfutil device --version
```

目標輸出約為：

```text
nrfutil 8.2.0 …
nrfutil-device 2.19.1 …
```

若 core 版本低於 8.2.x，先依官方文件升級 nrfutil core，再執行 `nrfutil install device`（或視當下官方文件用 `nrfutil upgrade`）。

版本管理常用指令：

| 指令 | 作用 |
|------|------|
| `nrfutil list` | 列出已安裝子命令與版本 |
| `nrfutil search` | 列出可安裝子命令 |
| `nrfutil install device` | 安裝 device；可指定版本，如 `nrfutil install device=2.19.1` |
| `nrfutil upgrade` / `nrfutil upgrade device` | 升級全部／指定子命令到最新 |
| `nrfutil self-upgrade` | 升級 core；`--to-version 8.2.0` 可指定或降回特定 core 版 |

### A.3 安裝 SEGGER J-Link Software

1. 下載總覽：[SEGGER J-Link downloads](https://www.segger.com/downloads/jlink/)，接受 SEGGER 授權後下載 **Software and Documentation Pack**。
2. 版本以 A.1 節官方指令查出的 tested 版為準（目前 device 2.19.1 對應 V9.24a）；較新版通常可用，異常時再切回 tested 版。

| 平台 | 安裝後應能在 PATH 找到 |
|------|------------------------|
| Windows | `JLink.exe`（或安裝程式加入 PATH 的同等命令） |
| macOS | `JLinkExe` |

Windows 安裝時請勾選把 J-Link 加入 PATH，或手動把安裝目錄加入系統 PATH。nRF Factory 啟動時的 preflight 檢查會尋找 `JLinkExe`（Unix）或 `JLink`（Windows 常見名）。

mac 開發機可選：`brew install --cask segger-jlink`；**Windows 產線一律用 SEGGER 官網安裝包**，不要用第三方套件管理工具。

### A.4 安裝後驗證（必跑）

在**同一台**將跑 nRF Factory 的機器上，接上（僅接）一顆 J-Link 後執行：

```bash
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
```

Windows（cmd / PowerShell，實際命令名依安裝結果為準）：

```text
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
where JLink
JLink
```

通過條件：

1. core / device 版本與 A.1 節一致或可接受（建議先對齊 8.2.0 + device 2.19.1）。
2. J-Link 為 nrfutil tested 版（現為 V9.24a）或經實測可用的較新版。
3. 只接一顆 probe 時，`device list` 能列出該 J-Link（serial 非空、`jlink: true`）。
4. 啟動 nRF Factory 後右上為**綠燈**，並顯示三項版本字串。

nrfutil 與 J-Link **無法**由 nRF Factory 一鍵代裝（授權與重散布限制），必須依本節手動安裝。

---

## B. 執行

### B.1 release zip 內容與解壓後結構

release zip 解壓後的結構與原始碼倉庫一致：執行檔在 `dist/`，文件在根層（`dist/nrf-factory-<version>-<os>-<arch>.zip` 展開後即得此佈局）。

```text
CHANGELOG.md
docs/INSTALL-AND-RUN.md
dist/
  nrf-factory-windows-amd64.exe      （Windows 版才有）
  nrf-factory-darwin-arm64           （macOS 版才有）
```

### B.2 啟動

執行檔為單一自足程式：啟動後綁定本機 loopback（`127.0.0.1`），並自動以獨立 profile 的 Chrome / Edge app 視窗開啟操作畫面。**關閉該 app 視窗，或在主控台按 Ctrl+C，都會停止服務。**

雙擊啟動：

| 平台 | 動作 |
|------|------|
| Windows | 進 `dist\` 資料夾，檔案總管雙擊 `nrf-factory-windows-amd64.exe` |
| macOS | Finder 雙擊 `dist/nrf-factory-darwin-arm64`（Unix 執行檔，Finder 會用終端機執行；首次可能被 Gatekeeper 擋，見下方處理方式） |

也可在終端機直接執行（以 macOS 為例）：

```bash
./dist/nrf-factory-darwin-arm64
```

啟動行為：

1. 綁定 `127.0.0.1` 隨機 port（不對外網開放），主控台印出 `nRF Factory: http://127.0.0.1:<port>`。
2. 找到 Chrome / Edge / Chromium / Brave 其一時，開獨立 app 視窗（專用 profile）；找不到時退回開系統預設瀏覽器分頁。
3. 關閉 app 視窗即停止服務；Ctrl+C 亦停止服務並關閉該 app 視窗。

啟動旗標與環境變數：

| 旗標 / 變數 | 說明 |
|-------------|------|
| `-addr` | loopback 監聽位址，預設 `127.0.0.1:0`（隨機 port）；工廠可固定，如 `127.0.0.1:17832` |
| `-no-browser` | 只啟動服務、不開瀏覽器；用主控台印出的網址自行開啟，Ctrl+C 停止 |
| `-version` | 印出版本後結束程式，不開瀏覽器 |
| `NRFUTIL_PATH` | `nrfutil` 不在 PATH 時，指向其完整路徑 |
| `NRF_FACTORY_PROFILE` | 覆寫 app 視窗使用的 Chrome profile 目錄（預設在系統暫存區） |

固定 port 範例：

```bash
./dist/nrf-factory-darwin-arm64 -addr 127.0.0.1:17832
```

需已安裝 Chrome、Edge、Chromium 或 Brave 其一才會走 app 視窗；否則自動退回一般瀏覽器分頁。

#### macOS Gatekeeper

未簽章、未 notarize 的執行檔從下載或 AirDrop 複製到別台 Mac 時，可能被 Gatekeeper 擋下：

```bash
xattr -d com.apple.quarantine nrf-factory-darwin-arm64
```

或在 Finder 中右鍵點擊執行檔後選「開啟」。

### B.3 狀態燈與畫面操作

啟動後確認右上角狀態點：

- **綠燈**：nrfutil、`device` 命令、J-Link 三者皆偵測正常，顯示類似 `nrfutil … · nrfutil-device … · J-Link OK`。
- **紅燈**：缺 nrfutil、缺 `device` 命令、或缺 J-Link，畫面會標明缺哪一項；依訊息安裝後**重開程式**。

正常燒錄操作順序：

1. 確認狀態為綠燈。
2. 點選 **.hex** 韌體檔（僅支援 `.hex`；檔案只在本次燒錄暫存，結束後刪除，不會寫進程式或程式目錄）。
3. 選擇 **L** 或 **R**（產品側別與成功計數分類；不對應不同 probe）。
4. 確認「開始燒錄 L/R」按鈕可按後點下。
5. 右側 terminal 即時串流 nrfutil 輸出：一般過程白字；成功綠字（該側計數 +1）；失敗紅字（計數不增加）。
6. **清除計數**：L/R 成功次數同時歸零（僅存在記憶體，重開程式也會歸零）。

燒錄進行中會鎖定檔案選擇、側別切換與清除計數，不可並行第二筆。

**注意**：燒錄會改寫目標 flash 並重置晶片。接上真機前請務必確認 `.hex` 與側別（L/R）正確。

### B.4 常見問題

| 現象 | 可能原因 | 處理 |
|------|----------|------|
| 狀態紅：找不到 nrfutil | 未安裝或未進 PATH | 依 A.2 節安裝；或設定 `NRFUTIL_PATH` |
| 狀態紅：缺少 device command | 只裝了 core | `nrfutil install device` |
| 狀態紅：找不到 J-Link | 未裝 J-Link Software | 依 A.3 節安裝並確認 PATH |
| 燒錄失敗：找不到 J-Link | USB 未接或驅動異常 | 重新插拔、以 `nrfutil device list --traits jlink` 檢查 |
| 燒錄失敗：找到多個 J-Link | 同時接了兩顆以上 probe | 只留一顆再燒 |
| 開始燒錄按鈕是灰的 | 尚未選 `.hex`，或正在燒錄中 | 先選檔；等目前燒錄結束 |
| 非 `.hex` 檔被拒 | 目前版本僅收 `.hex` | 換成 `.hex` 檔 |
| macOS 執行檔打不開 | Gatekeeper quarantine | 見 B.2 節 `xattr` 或右鍵開啟 |
| 關掉 app 視窗後服務就停了 | 這是預設行為（關窗即停服務） | 想保留服務可改用 `-no-browser`，再自行開啟印出的網址 |

---

安全與範圍：nRF Factory 只聽 loopback、不開對外 port；不提供 recover 或獨立 mass erase；不嵌入韌體、nrfutil、J-Link；同時只允許一個燒錄工作進行。
