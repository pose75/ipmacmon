# IPMAC 建置說明

## 快速建置

執行 `build.bat` 即可自動建置：

```cmd
build.bat
```

## 功能說明

- **自動包含模板檔案**：使用 Go 1.16+ 的 `embed` 功能，將 `templates` 目錄中的所有檔案嵌入到執行檔中
- **單一執行檔**：建置完成後只需要 `ipmac.exe` 一個檔案即可運行
- **自動相依性管理**：自動執行 `go mod tidy` 確保套件完整
- **檔案大小優化**：使用 `-ldflags="-s -w"` 減少執行檔大小

## 手動建置步驟

如果不使用 bat 檔案，可以手動執行以下步驟：

### 1. 下載相依套件
```cmd
go mod tidy
```

### 2. 建置執行檔
```cmd
go build -ldflags="-s -w" -o ipmac.exe .
```

## 建置需求

- **Go 版本**：1.16 或以上（需要 embed 功能支援）
- **作業系統**：Windows（bat 檔案為 Windows 專用）
- **架構**：AMD64

## 檔案結構

建置完成後的檔案結構：
```
ipmac/
├── ipmac.exe          # 主執行檔（包含所有模板檔案）
├── build.bat          # 建置腳本
├── go.mod             # Go 模組定義
├── go.sum             # Go 模組檢驗和
├── main.go            # 主程式進入點
├── *.go               # 其他 Go 原始檔案
└── templates/         # 模板檔案（會被嵌入到執行檔中）
    ├── index.html
    └── partials/
```

## 部署說明

建置完成後，只需要將 `ipmac.exe` 複製到目標機器即可運行，不需要：
- Go 運行環境
- templates 目錄
- 其他相依檔案

## 執行

```cmd
ipmac.exe
```

服務將在 `http://localhost:9999` 啟動。

## 疑難排解

### 問題：找不到模板檔案
**原因**：可能是 embed 路徑設定錯誤  
**解決**：確認 `//go:embed templates/*` 的路徑正確

### 問題：Go 版本過舊
**原因**：embed 功能需要 Go 1.16+  
**解決**：升級 Go 版本到 1.16 或以上

### 問題：建置失敗
**原因**：相依套件問題  
**解決**：執行 `go mod tidy` 重新下載套件