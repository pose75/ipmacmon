package main

import (
	"gorm.io/gorm/logger"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDatabase() {
	var err error

	// 連接SQLite資料庫
	db, err = gorm.Open(sqlite.Open("ipmac.db"), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
		Logger: logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		log.Fatal("連接資料庫失敗:", err)
	}

	// 自動遷移資料表
	err = db.AutoMigrate(&ArpEntry{}, &IPMacMapping{}, &NetworkConfig{})
	if err != nil {
		log.Fatal("資料表遷移失敗:", err)
	}

	// 檢查並更新現有的ArpEntry記錄的ScanBatch欄位
	updateExistingArpEntries()

	// 初始化預設網路設定
	initDefaultConfig()

	log.Println("資料庫初始化完成")
}

func initDefaultConfig() {
	var config NetworkConfig

	// 檢查是否已有設定
	result := db.First(&config)
	if result.Error == gorm.ErrRecordNotFound {
		// 建立預設設定
		defaultConfig := NetworkConfig{
			NetworkCIDR:      "10.2.10.0/24",
			TCPPorts:         "515,9100",
			ScanMethod:       "arp-scan", // 預設使用ARP掃描方式
			AutoScanEnabled:  false,      // 預設為手動掃描
			AutoScanInterval: 6,          // 預設6分鐘間隔
			SMTPServer:       "",         // 預設無SMTP伺服器
			SMTPPort:         465,        // 預設SMTP連接埠
			SMTPUsername:     "",         // 預設無使用者名稱
			SMTPPassword:     "",         // 預設無密碼
			MailTo:           "",         // 預設無收件者
		}

		if err := db.Create(&defaultConfig).Error; err != nil {
			log.Printf("建立預設設定失敗: %v", err)
		} else {
			log.Println("建立預設網路設定")
		}
	}
}

func updateExistingArpEntries() {
	// 直接嘗試更新空的scan_batch欄位，如果欄位不存在會被忽略
	var count int64
	result := db.Model(&ArpEntry{}).Where("scan_batch = '' OR scan_batch IS NULL").Count(&count)

	if result.Error == nil && count > 0 {
		log.Printf("發現 %d 筆舊記錄需要更新ScanBatch欄位", count)

		// 將所有空的scan_batch更新為 'legacy'
		updateResult := db.Model(&ArpEntry{}).
			Where("scan_batch = '' OR scan_batch IS NULL").
			Update("scan_batch", "legacy")

		if updateResult.Error != nil {
			log.Printf("更新舊記錄失敗: %v", updateResult.Error)
		} else {
			log.Printf("已更新 %d 筆舊記錄", updateResult.RowsAffected)
		}
	}
}
