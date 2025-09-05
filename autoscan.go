package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// AutoScanService 自動掃描服務
type AutoScanService struct {
	isRunning bool
	ticker    *time.Ticker
	stopCh    chan bool
	mutex     sync.Mutex
}

var autoScanService = &AutoScanService{
	stopCh: make(chan bool),
}

// StartAutoScan 開始自動掃描
func (s *AutoScanService) StartAutoScan() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.isRunning {
		return fmt.Errorf("自動掃描已在運行中")
	}
	
	// 讀取配置
	var config NetworkConfig
	if err := db.First(&config).Error; err != nil {
		return fmt.Errorf("無法載入網路設定: %v", err)
	}
	
	if !config.AutoScanEnabled {
		return fmt.Errorf("自動掃描未啟用，請先在設定中啟用")
	}
	
	// 檢查郵件設定完整性
	if !isMailConfigComplete(config) {
		return fmt.Errorf("郵件設定不完整，請先完成郵件設定")
	}
	
	// 創建定時器
	duration := time.Duration(config.AutoScanInterval) * time.Minute
	s.ticker = time.NewTicker(duration)
	s.isRunning = true
	
	// 啟動掃描協程
	go s.scanLoop(config)
	
	log.Printf("自動掃描已啟動，間隔: %d 分鐘", config.AutoScanInterval)
	return nil
}

// StopAutoScan 停止自動掃描
func (s *AutoScanService) StopAutoScan() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.isRunning {
		return fmt.Errorf("自動掃描未在運行")
	}
	
	s.stopCh <- true
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.isRunning = false
	
	log.Println("自動掃描已停止")
	return nil
}

// IsRunning 檢查是否正在運行
func (s *AutoScanService) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// scanLoop 掃描循環
func (s *AutoScanService) scanLoop(config NetworkConfig) {
	// 立即執行第一次掃描
	s.performScheduledScan(config)
	
	for {
		select {
		case <-s.ticker.C:
			// 定時掃描
			s.performScheduledScan(config)
			
		case <-s.stopCh:
			// 停止信號
			return
		}
	}
}

// performScheduledScan 執行定期掃描
func (s *AutoScanService) performScheduledScan(config NetworkConfig) {
	log.Println("開始執行自動掃描...")
	
	// 執行網路掃描
	entries, err := performNetworkScan(config)
	if err != nil {
		log.Printf("自動掃描失敗: %v", err)
		return
	}
	
	// 儲存掃描結果
	for _, entry := range entries {
		db.Create(&entry)
	}
	
	log.Printf("自動掃描完成，發現 %d 個設備", len(entries))
	
	// 檢查MAC地址變動並發送通知
	s.checkMACChangesAndNotify(entries, config)
}

// checkMACChangesAndNotify 檢查MAC地址變動並發送郵件通知
func (s *AutoScanService) checkMACChangesAndNotify(currentEntries []ArpEntry, config NetworkConfig) {
	// 讀取維護資料庫中的記錄
	var maintainedMappings []IPMacMapping
	db.Find(&maintainedMappings)
	
	// 建立維護資料庫的 IP->MAC 映射
	maintainedMap := make(map[string]IPMacMapping)
	for _, mapping := range maintainedMappings {
		maintainedMap[mapping.IP] = mapping
	}
	
	var changes []MACChange
	
	// 檢查MAC地址變動
	for _, entry := range currentEntries {
		if maintained, exists := maintainedMap[entry.IP]; exists {
			if strings.ToUpper(maintained.MAC) != strings.ToUpper(entry.MAC) {
				change := MACChange{
					IP:           entry.IP,
					OldMAC:       maintained.MAC,
					NewMAC:       entry.MAC,
					DeviceName:   maintained.Name,
					Description:  maintained.Description,
					DetectedTime: time.Now(),
				}
				changes = append(changes, change)
				
				log.Printf("檢測到MAC地址變動: IP %s, 舊MAC: %s, 新MAC: %s", 
					entry.IP, maintained.MAC, entry.MAC)
			}
		}
	}
	
	// 如果有變動，發送郵件通知
	if len(changes) > 0 {
		s.sendMACChangeNotification(changes, config)
	}
}

// MACChange MAC地址變動結構
type MACChange struct {
	IP           string
	OldMAC       string
	NewMAC       string
	DeviceName   string
	Description  string
	DetectedTime time.Time
}

// sendMACChangeNotification 發送MAC地址變動通知郵件
func (s *AutoScanService) sendMACChangeNotification(changes []MACChange, config NetworkConfig) {
	mailConfig := MailConfig{
		SMTPServer:   config.SMTPServer,
		SMTPPort:     config.SMTPPort,
		SMTPUsername: config.SMTPUsername,
		SMTPPassword: config.SMTPPassword,
		MailTo:       config.MailTo,
	}
	
	subject := fmt.Sprintf("IPMAC 系統警報：檢測到 %d 個MAC地址變動", len(changes))
	body := s.generateMACChangeEmailBody(changes)
	
	if err := SendNotificationMail(mailConfig, subject, body); err != nil {
		log.Printf("發送MAC變動通知郵件失敗: %v", err)
	} else {
		log.Printf("已發送MAC變動通知郵件，變動數量: %d", len(changes))
	}
}

// generateMACChangeEmailBody 生成MAC地址變動通知郵件內容
func (s *AutoScanService) generateMACChangeEmailBody(changes []MACChange) string {
	body := fmt.Sprintf(`
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .change-item { background-color: #f8f9fa; margin: 10px 0; padding: 15px; border-left: 4px solid #dc3545; }
        .time { color: #666; font-size: 12px; }
        table { width: 100%%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h2>🚨 IPMAC 系統警報</h2>
        <p>檢測到網路設備MAC地址變動</p>
    </div>
    
    <div class="content">
        <p><strong>檢測時間：</strong> %s</p>
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
        
        <div style="margin-top: 30px; padding: 15px; background-color: #fff3cd; border: 1px solid #ffeaa7; border-radius: 5px;">
            <h4>⚠️ 注意事項</h4>
            <ul>
                <li>MAC地址變動可能表示網路設備已更換</li>
                <li>請檢查變動的設備是否為預期的變更</li>
                <li>如有異常，請立即檢查網路安全</li>
                <li>可在系統中更新維護清單以反映變更</li>
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

// isMailConfigComplete 檢查郵件設定是否完整
func isMailConfigComplete(config NetworkConfig) bool {
	return config.SMTPServer != "" && 
		   config.SMTPUsername != "" && 
		   config.SMTPPassword != "" && 
		   config.MailTo != ""
}