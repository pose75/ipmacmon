package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// 首頁
func indexHandler(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "IP/MAC 記錄工具",
	})
}

// ArpEntryWithName ARP記錄含設備名稱
type ArpEntryWithName struct {
	ArpEntry
	DeviceName string `json:"device_name"`
}

// 取得ARP掃描記錄
func getArpEntriesHandler(c *fiber.Ctx) error {
	var entries []ArpEntry

	scanBatch := c.Query("batch")

	if scanBatch != "" {
		// 取得指定批次的記錄，按IP排序
		db.Where("scan_batch = ?", scanBatch).Order("ip").Find(&entries)
	} else {
		// 取得最新批次的記錄
		var latestBatch string
		db.Model(&ArpEntry{}).Select("scan_batch").Order("timestamp desc").Limit(1).Scan(&latestBatch)

		if latestBatch != "" {
			db.Where("scan_batch = ?", latestBatch).Order("ip").Find(&entries)
		}
	}

	// 獲取維護表中的設備名稱
	var mappings []IPMacMapping
	db.Find(&mappings)
	
	// 建立IP->名稱映射
	nameMap := make(map[string]string)
	for _, mapping := range mappings {
		if mapping.Name != "" {
			nameMap[mapping.IP] = mapping.Name
		}
	}
	
	// 組合結果
	var result []ArpEntryWithName
	for _, entry := range entries {
		entryWithName := ArpEntryWithName{
			ArpEntry:   entry,
			DeviceName: nameMap[entry.IP],
		}
		result = append(result, entryWithName)
	}

	return c.JSON(result)
}

// 取得掃描批次清單
func getScanBatchesHandler(c *fiber.Ctx) error {
	var batches []struct {
		ScanBatch string `json:"scan_batch"`
		Count     int64  `json:"count"`
		Timestamp string `json:"timestamp"`
	}

	// 取得最近10次掃描批次
	db.Model(&ArpEntry{}).
		Select("scan_batch, COUNT(*) as count, MAX(timestamp) as timestamp").
		Group("scan_batch").
		Order("MAX(timestamp) desc").
		Limit(10).
		Scan(&batches)

	return c.JSON(batches)
}

// 取得IP/MAC對應維護記錄
func getIPMacMappingsHandler(c *fiber.Ctx) error {
	var mappings []IPMacMapping

	// 取得排序參數，預設按IP位址排序
	sortBy := c.Query("sort", "ip")

	var orderClause string
	switch sortBy {
	case "ip":
		// 簡單的IP字串排序，雖然不完美但足夠使用
		orderClause = "ip"
	default:
		orderClause = "updated_at desc"
	}

	db.Order(orderClause).Find(&mappings)

	return c.JSON(mappings)
}

// 取得網路設定
func getNetworkConfigHandler(c *fiber.Ctx) error {
	var config NetworkConfig

	if err := db.First(&config).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "設定未找到"})
	}

	return c.JSON(config)
}

// 更新網路設定
func updateNetworkConfigHandler(c *fiber.Ctx) error {
	var req struct {
		NetworkCIDR string `json:"network_cidr"`
		TCPPorts    string `json:"tcp_ports"`
		ScanMethod  string `json:"scan_method"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}
	
	// 驗證 CIDR 格式
	if !isValidCIDR(req.NetworkCIDR) {
		return c.Status(400).JSON(fiber.Map{"error": "CIDR 格式錯誤，請使用正確格式如: 10.2.10.0/24"})
	}

	var config NetworkConfig
	result := db.First(&config)

	if result.Error != nil {
		// 如果沒有設定，建立新的
		config = NetworkConfig{
			NetworkCIDR: req.NetworkCIDR,
			TCPPorts:    req.TCPPorts,
			ScanMethod:  req.ScanMethod,
		}
		db.Create(&config)
	} else {
		// 更新現有設定
		config.NetworkCIDR = req.NetworkCIDR
		config.TCPPorts = req.TCPPorts
		config.ScanMethod = req.ScanMethod
		db.Save(&config)
	}

	return c.JSON(fiber.Map{"message": "設定已更新"})
}

// 執行掃描
func scanHandler(c *fiber.Ctx) error {
	// 讀取網路設定
	var config NetworkConfig
	if err := db.First(&config).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "無法載入網路設定"})
	}

	// 執行真實的網路掃描
	entries, err := performNetworkScan(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("掃描失敗: %v", err),
		})
	}

	// 將掃描結果存入資料庫
	for _, entry := range entries {
		db.Create(&entry)
	}
	
	// 比較掃描結果與維護資料庫中的記錄
	var maintainedMappings []IPMacMapping
	db.Find(&maintainedMappings)
	
	// 建立維護資料庫的 IP->MAC 映射
	maintainedMap := make(map[string]string)
	maintainedNameMap := make(map[string]string)
	for _, mapping := range maintainedMappings {
		maintainedMap[mapping.IP] = mapping.MAC
		maintainedNameMap[mapping.IP] = mapping.Name
	}
	
	// 找出不符合的項目
	var mismatches []fiber.Map
	for _, entry := range entries {
		if maintainedMAC, exists := maintainedMap[entry.IP]; exists {
			// IP 在維護資料庫中存在，檢查 MAC 是否相符
			if strings.ToUpper(maintainedMAC) != strings.ToUpper(entry.MAC) {
				mismatch := fiber.Map{
					"ip":             entry.IP,
					"scanned_mac":    entry.MAC,
					"maintained_mac": maintainedMAC,
					"type":          "mac_mismatch",
				}
				if name, hasName := maintainedNameMap[entry.IP]; hasName && name != "" {
					mismatch["name"] = name
				}
				mismatches = append(mismatches, mismatch)
			}
		}
	}
	
	response := fiber.Map{
		"message": "掃描完成",
		"count":   len(entries),
		"config": fiber.Map{
			"network_cidr": config.NetworkCIDR,
			"tcp_ports":    config.TCPPorts,
			"scan_method":  config.ScanMethod,
		},
	}
	
	if len(mismatches) > 0 {
		response["mismatches"] = mismatches
		response["mismatch_count"] = len(mismatches)
		
		// 如果有MAC地址變動且郵件設定完整，發送通知郵件
		if isMailConfigComplete(config) {
			// 轉換mismatch格式為MACChange格式
			var changes []MACChange
			for _, mismatch := range mismatches {
				change := MACChange{
					IP:           mismatch["ip"].(string),
					OldMAC:       mismatch["maintained_mac"].(string),
					NewMAC:       mismatch["scanned_mac"].(string),
					DeviceName:   "",
					Description:  "",
					DetectedTime: time.Now(),
				}
				
				if name, exists := mismatch["name"]; exists {
					change.DeviceName = name.(string)
				}
				
				// 獲取設備說明
				var mapping IPMacMapping
				if err := db.Where("ip = ?", change.IP).First(&mapping).Error; err == nil {
					change.Description = mapping.Description
				}
				
				changes = append(changes, change)
			}
			
			// 發送郵件通知
			go sendManualScanMACChangeNotification(changes, config)
		}
	}
	
	return c.JSON(response)
}

// 新增或更新IP/MAC對應
func updateIPMacMappingHandler(c *fiber.Ctx) error {
	var req struct {
		IP          string `json:"ip"`
		MAC         string `json:"mac"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}
	
	// 驗證 MAC 地址格式
	if !isValidMAC(req.MAC) {
		return c.Status(400).JSON(fiber.Map{"error": "MAC 地址格式錯誤，請使用正確格式如: AA:BB:CC:DD:EE:FF"})
	}
	
	// 標準化 MAC 地址格式
	req.MAC = normalizeMAC(req.MAC)

	var mapping IPMacMapping
	result := db.Where("ip = ?", req.IP).First(&mapping)

	if result.Error != nil {
		// 新增
		mapping = IPMacMapping{
			IP:          req.IP,
			MAC:         req.MAC,
			Name:        req.Name,
			Description: req.Description,
		}
		db.Create(&mapping)
	} else {
		// 更新
		mapping.MAC = req.MAC
		mapping.Name = req.Name
		mapping.Description = req.Description
		db.Save(&mapping)
	}

	return c.JSON(fiber.Map{"message": "IP/MAC對應已更新"})
}

// 檢查IP是否已存在
func checkIPExistsHandler(c *fiber.Ctx) error {
	ip := c.Params("ip")

	var mapping IPMacMapping
	result := db.Where("ip = ?", ip).First(&mapping)

	if result.Error != nil {
		// IP不存在
		return c.JSON(fiber.Map{
			"exists": false,
		})
	}

	// IP已存在，回傳現有資料
	return c.JSON(fiber.Map{
		"exists":      true,
		"current_mac": mapping.MAC,
		"name":        mapping.Name,
		"description": mapping.Description,
		"updated_at":  mapping.UpdatedAt,
	})
}

// 刪除IP/MAC對應
func deleteIPMacMappingHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "無效的ID"})
	}

	result := db.Delete(&IPMacMapping{}, id)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "刪除失敗"})
	}

	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "記錄未找到"})
	}

	return c.JSON(fiber.Map{"message": "已刪除"})
}

// 清除所有IP/MAC對應
func clearAllIPMacMappingsHandler(c *fiber.Ctx) error {
	result := db.Where("1 = 1").Delete(&IPMacMapping{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "清除失敗"})
	}

	return c.JSON(fiber.Map{
		"message":       "所有記錄已清除",
		"deleted_count": result.RowsAffected,
	})
}

// 取得自動掃描設定
func getAutoScanConfigHandler(c *fiber.Ctx) error {
	var config NetworkConfig

	if err := db.First(&config).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "設定未找到"})
	}

	return c.JSON(fiber.Map{
		"auto_scan_enabled":  config.AutoScanEnabled,
		"auto_scan_interval": config.AutoScanInterval,
	})
}

// 更新自動掃描設定
func updateAutoScanConfigHandler(c *fiber.Ctx) error {
	var req struct {
		AutoScanEnabled  bool `json:"auto_scan_enabled"`
		AutoScanInterval int  `json:"auto_scan_interval"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}

	// 驗證間隔時間
	if req.AutoScanEnabled && (req.AutoScanInterval < 1 || req.AutoScanInterval > 1440) {
		return c.Status(400).JSON(fiber.Map{"error": "掃描間隔必須在 1-1440 分鐘之間"})
	}

	var config NetworkConfig
	result := db.First(&config)

	if result.Error != nil {
		// 如果沒有設定，建立新的
		config = NetworkConfig{
			NetworkCIDR:      "10.2.10.0/24", // 默認值
			TCPPorts:         "515,9100",      // 默認值
			ScanMethod:       "arp-scan",      // 默認值
			AutoScanEnabled:  req.AutoScanEnabled,
			AutoScanInterval: req.AutoScanInterval,
		}
		db.Create(&config)
	} else {
		// 更新現有設定
		config.AutoScanEnabled = req.AutoScanEnabled
		config.AutoScanInterval = req.AutoScanInterval
		db.Save(&config)
	}

	return c.JSON(fiber.Map{"message": "自動掃描設定已更新"})
}

// 取得郵件設定
func getMailConfigHandler(c *fiber.Ctx) error {
	var config NetworkConfig

	if err := db.First(&config).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "設定未找到"})
	}

	return c.JSON(fiber.Map{
		"smtp_server":   config.SMTPServer,
		"smtp_port":     config.SMTPPort,
		"smtp_username": config.SMTPUsername,
		"smtp_password": config.SMTPPassword,
		"mail_to":       config.MailTo,
	})
}

// 更新郵件設定
func updateMailConfigHandler(c *fiber.Ctx) error {
	var req struct {
		SMTPServer   string `json:"smtp_server"`
		SMTPPort     int    `json:"smtp_port"`
		SMTPUsername string `json:"smtp_username"`
		SMTPPassword string `json:"smtp_password"`
		MailTo       string `json:"mail_to"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}

	// 驗證連接埠範圍
	if req.SMTPPort < 1 || req.SMTPPort > 65535 {
		return c.Status(400).JSON(fiber.Map{"error": "SMTP 連接埠必須在 1-65535 之間"})
	}

	var config NetworkConfig
	result := db.First(&config)

	if result.Error != nil {
		// 如果沒有設定，建立新的
		config = NetworkConfig{
			NetworkCIDR:      "10.2.10.0/24", // 默認值
			TCPPorts:         "515,9100",      // 默認值
			ScanMethod:       "arp-scan",      // 默認值
			AutoScanEnabled:  false,           // 默認值
			AutoScanInterval: 6,               // 默認值
			SMTPServer:       req.SMTPServer,
			SMTPPort:         req.SMTPPort,
			SMTPUsername:     req.SMTPUsername,
			SMTPPassword:     req.SMTPPassword,
			MailTo:           req.MailTo,
		}
		db.Create(&config)
	} else {
		// 更新現有設定
		config.SMTPServer = req.SMTPServer
		config.SMTPPort = req.SMTPPort
		config.SMTPUsername = req.SMTPUsername
		config.SMTPPassword = req.SMTPPassword
		config.MailTo = req.MailTo
		db.Save(&config)
	}

	return c.JSON(fiber.Map{"message": "郵件設定已更新"})
}

// 測試郵件連接
func testMailHandler(c *fiber.Ctx) error {
	var req struct {
		SMTPServer   string `json:"smtp_server"`
		SMTPPort     int    `json:"smtp_port"`
		SMTPUsername string `json:"smtp_username"`
		SMTPPassword string `json:"smtp_password"`
		MailTo       string `json:"mail_to"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}

	// 基本驗證
	if req.SMTPServer == "" || req.SMTPUsername == "" || req.SMTPPassword == "" || req.MailTo == "" {
		return c.Status(400).JSON(fiber.Map{"error": "所有欄位都必須填寫"})
	}

	// 驗證連接埠範圍
	if req.SMTPPort < 1 || req.SMTPPort > 65535 {
		return c.Status(400).JSON(fiber.Map{"error": "SMTP 連接埠必須在 1-65535 之間"})
	}

	// 創建郵件配置
	mailConfig := MailConfig{
		SMTPServer:   req.SMTPServer,
		SMTPPort:     req.SMTPPort,
		SMTPUsername: req.SMTPUsername,
		SMTPPassword: req.SMTPPassword,
		MailTo:       req.MailTo,
	}

	// 先測試SMTP連接
	if err := TestSMTPConnection(mailConfig); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("SMTP 連接測試失敗: %v", err),
		})
	}

	// 發送測試郵件
	if err := SendTestMail(mailConfig); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("發送測試郵件失敗: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("✅ 郵件測試成功！\n\n伺服器: %s:%d\n收件者: %s\n\n測試郵件已發送，請檢查您的信箱。", 
			req.SMTPServer, req.SMTPPort, req.MailTo),
	})
}

// 自動掃描控制
func autoScanControlHandler(c *fiber.Ctx) error {
	var req struct {
		Action string `json:"action"` // "start" 或 "stop"
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "請求格式錯誤"})
	}

	switch req.Action {
	case "start":
		if err := autoScanService.StartAutoScan(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "自動掃描已啟動"})

	case "stop":
		if err := autoScanService.StopAutoScan(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "自動掃描已停止"})

	default:
		return c.Status(400).JSON(fiber.Map{"error": "無效的操作，請使用 'start' 或 'stop'"})
	}
}

// 取得自動掃描狀態
func autoScanStatusHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"is_running": autoScanService.IsRunning(),
	})
}

// sendManualScanMACChangeNotification 發送手動掃描MAC變動通知郵件
func sendManualScanMACChangeNotification(changes []MACChange, config NetworkConfig) {
	mailConfig := MailConfig{
		SMTPServer:   config.SMTPServer,
		SMTPPort:     config.SMTPPort,
		SMTPUsername: config.SMTPUsername,
		SMTPPassword: config.SMTPPassword,
		MailTo:       config.MailTo,
	}
	
	subject := fmt.Sprintf("IPMAC 系統通知：手動掃描發現 %d 個MAC地址變動", len(changes))
	body := generateManualScanMACChangeEmailBody(changes)
	
	if err := SendNotificationMail(mailConfig, subject, body); err != nil {
		log.Printf("發送手動掃描MAC變動通知郵件失敗: %v", err)
	} else {
		log.Printf("已發送手動掃描MAC變動通知郵件，變動數量: %d", len(changes))
	}
}

// generateManualScanMACChangeEmailBody 生成手動掃描MAC變動通知郵件內容
func generateManualScanMACChangeEmailBody(changes []MACChange) string {
	body := fmt.Sprintf(`
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background-color: #ffc107; color: #212529; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .change-item { background-color: #f8f9fa; margin: 10px 0; padding: 15px; border-left: 4px solid #ffc107; }
        .time { color: #666; font-size: 12px; }
        table { width: 100%%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h2>📋 IPMAC 系統通知</h2>
        <p>手動掃描發現網路設備MAC地址變動</p>
    </div>
    
    <div class="content">
        <p><strong>掃描時間：</strong> %s</p>
        <p><strong>掃描類型：</strong> 手動掃描</p>
        <p><strong>變動數量：</strong> %d 個設備</p>
        
        <h3>變動詳情：</h3>
        
        <table>
            <tr>
                <th>IP地址</th>
                <th>設備名稱</th>
                <th>原MAC地址</th>
                <th>新MAC地址</th>
                <th>說明</th>
            </tr>`, 
        time.Now().Format("2006-01-02 15:04:05"), len(changes))
	
	for _, change := range changes {
		deviceName := change.DeviceName
		if deviceName == "" {
			deviceName = "未設定"
		}
		description := change.Description
		if description == "" {
			description = "無"
		}
		
		body += fmt.Sprintf(`
            <tr>
                <td><strong>%s</strong></td>
                <td>%s</td>
                <td style="color: #dc3545;">%s</td>
                <td style="color: #28a745;">%s</td>
                <td>%s</td>
            </tr>`,
			change.IP, deviceName, change.OldMAC, change.NewMAC, description)
	}
	
	body += `
        </table>
        
        <div style="margin-top: 30px; padding: 15px; background-color: #d1ecf1; border: 1px solid #bee5eb; border-radius: 5px;">
            <h4>ℹ️ 掃描資訊</h4>
            <ul>
                <li>這是手動掃描的結果通知</li>
                <li>系統在網頁上也會顯示相同的變動資訊</li>
                <li>建議檢查變動是否符合預期</li>
                <li>如需要，可在系統中更新維護清單</li>
            </ul>
        </div>
        
        <hr>
        <p style="color: #666; font-size: 12px;">
            此郵件由 IPMAC 網路監控系統自動發送<br>
            系統時間：%s
        </p>
    </div>
</body>
</html>
	`
	
	body = fmt.Sprintf(body, time.Now().Format("2006-01-02 15:04:05"))
	return body
}

// 驗證 CIDR 格式
func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// 驗證 MAC 地址格式
func isValidMAC(mac string) bool {
	_, err := net.ParseMAC(mac)
	return err == nil
}

// 標準化 MAC 地址格式 (統一轉為大寫，使用冒號分隔)
func normalizeMAC(mac string) string {
	if hw, err := net.ParseMAC(mac); err == nil {
		return strings.ToUpper(hw.String())
	}
	return strings.ToUpper(mac)
}
