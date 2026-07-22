# 工站工具固定安裝（nRF Util / J-Link）

本文件鎖定**本專案開發機已驗證可跑**的工具版本，供換機、產線 PC 重裝時對齊。  
nRF Factory **不內嵌、不重散布** Nordic / SEGGER 軟體（各有授權）；工站須自行下載安裝。

## 已驗證版本（本機實測）

下列字串以開發機（macOS arm64）`nrfutil` / `JLinkExe` 實際輸出為準：

| 元件 | 版本 | 備註 |
|------|------|------|
| nRF Util（core） | **8.2.0**（commit `c910332`，2026-04-21） | `nrfutil --version` |
| nRF Util **device** command | **2.19.1**（commit `181e493`，2026-06-03） | `nrfutil device --version` |
| SEGGER J-Link Software | tested **V9.24a**（device 2.19.1 對應）；本機另裝 **V9.60** 亦實測可跑 | `JLinkExe` 啟動 banner；安裝目錄常見 `JLink_V9xx` |

**J-Link 目標版本以 nrfutil 官方輸出為準，不是某台機器現裝版。** device 2.19.1 對應的 tested 版為 **V9.24a**，用官方指令取得（機器無關）：

```bash
nrfutil device --version        # 文字同時印 tested 與本機現裝
nrfutil device --version --json | jq -r '.. | objects | select(.name=="JlinkARM") | .expectedVersion.version'
```

tested 版**非強制**：較新版（本機 V9.60 已實測 list / program 正常）通常照跑；只有出現 J-Link 相關異常，才切回 tested 版 V9.24a。

nRF Factory 右上狀態列成功時類似：

```text
nrfutil 8.2.0 … · nrfutil-device 2.19.1 … · J-Link OK
```

## 下載位置（官方）

### 1. nRF Util

- 產品頁：[nRF Util](https://www.nordicsemi.com/Products/Development-tools/nRF-Util)
- 安裝說明：[Installing nRF Util](https://docs.nordicsemi.com/bundle/nrfutil/page/guides/installing.html)

依 OS 下載官方安裝包（Windows / macOS / Linux）。  
Homebrew cask `nrfutil` 僅 mac 開發便利路徑，**產線 Windows 請走 Nordic 官網**，勿依賴 brew。

安裝 core 後，**必須**再裝 device 子命令（與本機對齊）：

```bash
nrfutil install device
# 若需對到與本機相同的 device 大版，可再查：
nrfutil list
nrfutil device --version
```

目標：

```text
nrfutil 8.2.0 …
nrfutil-device 2.19.1 …
```

若 core 低於 8.2.x，先依官方方式升級 nrfutil，再 `nrfutil install device`（或 `nrfutil upgrade`，以當下官方文件為準）。

### 2. SEGGER J-Link Software and Documentation Pack

- 下載總覽：[SEGGER J-Link downloads](https://www.segger.com/downloads/jlink/)
- 接受 SEGGER 授權後下載 **Software and Documentation Pack**
- 版本以 nrfutil 報的 tested 版為準（device 2.19.1 → **V9.24a**）；較新版（如本機 **V9.60**，banner `V9.60`）通常可用，異常時切回 tested 版

| 平台 | 安裝後應能在 PATH 找到 |
|------|------------------------|
| Windows | `JLink.exe`（或安裝程式加入 PATH 的同等命令） |
| macOS | `JLinkExe` |

Windows 安裝時勾選把 J-Link 加入 PATH，或手動把安裝目錄加入系統 PATH。  
本專案 preflight 會 `LookPath`：`JLinkExe`（Unix）/ `JLink`（Windows 常見名）。

mac 開發機可選：`brew install --cask segger-jlink`（曾裝到 9.60）；**Windows 產線用 SEGGER 官網安裝包**。

## 安裝後驗證（必跑）

在**同一台**將跑 nRF Factory 的機器上：

```bash
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
```

Windows（cmd / PowerShell，命令名依安裝為準）：

```text
nrfutil --version
nrfutil device --version
nrfutil device list --traits jlink --json
where JLink
JLink
```

通過條件：

1. core / device 版本與上表一致或可接受（建議先對齊 8.2.0 + device 2.19.1）。
2. J-Link 為 nrfutil tested 版（device 2.19.1 → **V9.24a**）或經實測可用的較新版（如 V9.60）。
3. `device list` 在只接**一顆** probe 時可列出該 J-Link（serial 非空、`jlink: true`）。
4. 啟動 nRF Factory 後右上 **綠燈** + 三項字串。

## Windows 10 / 11 產線建議順序

1. 安裝 **J-Link Software Pack**（版本對 `nrfutil device --version` 報的 tested 版，device 2.19.1 → **V9.24a**；較新版通常亦可）→ 確認 PATH。
2. 安裝 **nRF Util**（Nordic 官網）→ `nrfutil install device` → 核對 8.2.0 / 2.19.1。
3. 接一顆 J-Link → `nrfutil device list --traits jlink`。
4. 放置 `nrf-factory-windows-amd64.exe`（及韌體 `.hex`，外部選取）。
5. 執行程式 → 綠燈 → 選 hex → L/R 燒錄。

瀏覽器：建議 **Chrome 或 Edge**（若使用 app 窗啟動腳本）。  
nrfutil / J-Link **無法**由 nRF Factory 一鍵代裝（授權與重散布限制）。

## macOS 開發機（參考，與本機一致）

本機實際路徑範例（僅供對照，**勿寫進產線 SOP 當必填路徑**）：

- nrfutil：Homebrew cask 引導安裝後 self-contained CLI → 報告 **8.2.0**
- J-Link：`/Applications/SEGGER/JLink_V960`，`JLinkExe` → **V9.60**

驗證命令與上節相同。未簽章 binary 可能被 Gatekeeper 擋，與工具安裝無關。

## 版本漂移怎麼辦

| 情況 | 做法 |
|------|------|
| 現場 nrfutil 比 8.2.0 新，device 也新 | 先在備援機跑通 list/program 再換產線；通過則更新本文件版本列 |
| 現場只有舊 nrfutil | 升到 8.2.x + device 2.19.1 再燒 |
| J-Link 版本疑慮 | 以 `nrfutil device --version` 報的 tested 版為準（現 **V9.24a**）；較新版通常可用，異常時改裝 tested 版 |
| 多顆 J-Link | 拔到只剩一顆（本程式第一版不選 serial） |

**不要**把 nrfutil / J-Link 安裝包 commit 進本 git 倉庫（體積、授權、資安）。  
本文件只鎖定**版本號與官方下載入口**。

## 與 nRF Factory 的關係

| 項目 | 負責方 |
|------|--------|
| 燒錄 UI / 呼叫 `nrfutil device program` | nRF Factory binary |
| 安裝 nrfutil、device、J-Link | 工站 / IT（本文件） |
| `.hex` 韌體 | 產線選檔，不嵌入程式 |

缺工具時程式仍可開 UI，狀態列會紅字指出缺哪一項；燒錄會在裝置列舉或 program 階段失敗，且**不會**誤加成功計數。
