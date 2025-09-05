package main

import (
	"time"
)

// ArpEntry ARP表項記錄
type ArpEntry struct {
	ID        uint      `gorm:"primarykey"`
	IP        string    `gorm:"not null;index"`
	MAC       string    `gorm:"not null"`
	Timestamp time.Time `gorm:"not null;index"`
	ScanBatch string    `gorm:"default:'unknown';index"` // 掃描批次ID
	CreatedAt time.Time
}

// IPMacMapping IP和MAC的對應關係維護表
type IPMacMapping struct {
	ID          uint   `gorm:"primarykey"`
	IP          string `gorm:"uniqueIndex;not null"`
	MAC         string `gorm:"not null"`
	Name        string `gorm:"size:100"` // 使用者名稱或設備名稱
	Description string `gorm:"size:255"`
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

// NetworkConfig 網路掃描設定
type NetworkConfig struct {
	ID               uint   `gorm:"primarykey"`
	NetworkCIDR      string `gorm:"not null"`           // 如 10.2.10.0/24
	TCPPorts         string `gorm:"not null"`           // 如 80,443,8080
	ScanMethod       string `gorm:"default:'arp-scan'"` // 掃描方式: arp-scan, tcp, ping
	AutoScanEnabled  bool   `gorm:"default:false"`      // 是否啟用自動掃描
	AutoScanInterval int    `gorm:"default:60"`         // 自動掃描間隔 (分鐘)
	SMTPServer       string `gorm:"default:''"`         // SMTP 伺服器
	SMTPPort         int    `gorm:"default:587"`        // SMTP 連接埠
	SMTPUsername     string `gorm:"default:''"`         // SMTP 使用者名稱
	SMTPPassword     string `gorm:"default:''"`         // SMTP 密碼
	MailTo           string `gorm:"default:''"`         // 收件者信箱
	UpdatedAt        time.Time
	CreatedAt        time.Time
}
