# IPMAC - 網路設備 ARP 監控系統

一個基於 Go 語言開發的 ARP 表監控應用程序，用於追蹤網路設備的 MAC 地址並檢測變化。該應用程序提供網頁介面進行網路掃描和 MAC 地址管理。

## 主要功能

- **即時網路掃描** - 透過網頁介面進行即時網路掃描
- **網路範圍配置** - 支援 CIDR 格式配置（如：10.2.10.0/24）
- **TCP 埠配置** - 可指定掃描的 TCP 埠（如：80, 443, 8080）
- **三步驟掃描流程**：
  1. 清除本地 ARP 表
  2. 嘗試連接指定 TCP 埠
  3. 讀取更新後的 ARP 表
- **SQLite3 資料庫存儲** - 帶時間戳的歷史記錄
- **MAC 地址變更檢測與警報**
- **IP/MAC 維護介面**
- **自動掃描功能** - 可設定定時自動掃描
- **郵件通知** - MAC 地址變更時發送郵件警報

## 技術架構

- **後端框架**: Go + Fiber 網頁框架
- **資料庫**: SQLite3 + GORM ORM
- **前端模板**: HTML 模板引擎
- **支援平台**: Windows、Linux、macOS（主要針對 Windows 平台優化）

## 快速開始

### 環境需求

- Go 1.23.5 或更高版本
- Windows 系統（需要管理員權限執行）
- SQLite3 支援

### 安裝與執行

1. **下載執行檔**
   - 直接下載 `ipmac.exe` 執行檔即可使用
   - 無需額外安裝其他相依套件

2. **以管理員權限執行**
   ```
   右鍵點擊 ipmac.exe → 以系統管理員身分執行
   ```

3. **訪問網頁介面**
   - 開啟瀏覽器訪問：http://localhost:9999

## 使用步驟

### Step 1: 系統設定
- 進入「系統設定」功能
- 設定要掃描的網段 CIDR（如：192.168.1.0/24）
- **注意**：掃描時必須在同網段內（因為使用 ARP scan）
- 設定要掃描的 TCP 埠（如：80,443,8080）

### Step 2: 網路掃描
- 使用「網路掃描」功能進行掃描
- 掃描完成後，在「掃描記錄」中查看結果
- 可將掃描結果匯入到「IP/MAC 維護」表中

### Step 3: IP/MAC 維護
- 在「IP/MAC 維護」中管理設備資訊
- 為每筆記錄添加註解（如：2F茶水間印表機）
- 方便記憶每個 IP 的用途和設備名稱

### Step 4: 變更監控
- 下次進行「網路掃描」時
- 若發現 IP 和 MAC 有變動，系統會顯示警告訊息
- 支援 MAC 地址變動時自動發送郵件通知

## 開發相關

### 開發環境建置

```bash
# 克隆專案
git clone <repository-url>
cd ipmac

# 安裝相依套件
go mod download
go mod tidy

# 建置應用程序
go build

# 執行應用程序
go run main.go
```

### 專案結構

```
ipmac/
├── main.go              # 應用程序入口點
├── models.go            # 資料模型定義
├── database.go          # 資料庫初始化和操作
├── scanner.go           # 網路掃描核心功能
├── handlers.go          # HTTP 請求處理器
├── routes.go            # 路由配置
├── autoscan.go          # 自動掃描功能
├── mail.go              # 郵件通知功能
├── admin_*.go           # 各平台管理員權限檢查
├── templates/           # HTML 模板文件
├── go.mod               # Go 模組定義
└── README.md            # 專案說明文件
```

### 跨平台建置

```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o bin/ipmac-linux-amd64

# Linux (arm64)
GOOS=linux GOARCH=arm64 go build -o bin/ipmac-linux-arm64

# macOS (amd64)
GOOS=darwin GOARCH=amd64 go build -o ipmac-darwin-amd64

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o bin/ipmac-windows-amd64.exe

# Windows (arm64)
GOOS=windows GOARCH=arm64 go build -o bin/ipmac-windows-arm64.exe
```

## 掃描原理

本應用程序採用特殊的掃描方式，不依賴傳統的 ARP 掃描工具：

1. **清除 ARP 表**：使用 `netsh interface ip delete arpcache` 清空本地 ARP 快取
2. **TCP 連接嘗試**：對目標 IP 範圍的指定埠進行 TCP 連接
3. **讀取 ARP 表**：使用 `arp -a` 命令讀取更新後的 ARP 表

這種方式的優點：
- 不需要額外的網路掃描工具
- 可以繞過某些網路安全限制
- 結果更加準確，只顯示真實存在的設備

## 資料庫結構

### ArpEntry（ARP 記錄表）
- 儲存 IP 地址和 MAC 地址的掃描記錄
- 包含時間戳和掃描批次信息
- 用於追蹤網路設備的歷史狀態

### IPMacMapping（IP-MAC 對應表）
- 維護 IP 和 MAC 地址的對應關係
- 支援設備名稱和描述信息
- 用於識別已知設備和檢測變更

### NetworkConfig（網路配置表）
- 網路掃描參數配置
- 自動掃描設定
- SMTP 郵件伺服器配置

## 注意事項

- **管理員權限**：程序必須以管理員權限執行才能清除 ARP 表和執行網路操作
- **網路環境**：掃描時必須在同一網段內，因為使用 ARP 掃描原理
- **防火牆設定**：確保防火牆允許程序的網路活動
- **掃描頻率**：避免過於頻繁的掃描，以免對網路造成影響
- **系統相容性**：主要針對 Windows 平台優化，其他平台功能可能有限

## 系統特色

- **非資通系統**：在個人電腦即可執行的小工具，不需要系統分級
- **簡單易用**：只需下載執行檔即可使用，無需複雜安裝程序
- **網頁介面**：直覺的網頁操作介面，易於使用和管理
- **資料持久化**：所有掃描記錄和設定都存儲在本地 SQLite 資料庫中
- **變更監控**：自動檢測網路設備變更並提供警告和通知功能

## 貢獻

歡迎提交 Issue 和 Pull Request 來改進這個專案。

## 授權

請查看 LICENSE 文件了解授權信息。