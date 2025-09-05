package main

import (
	"crypto/tls"
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

// MailConfig 郵件配置結構
type MailConfig struct {
	SMTPServer   string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	MailTo       string
}

// SendTestMail 發送測試郵件
func SendTestMail(config MailConfig) error {
	m := gomail.NewMessage()
	
	// 設定發送者
	m.SetHeader("From", config.SMTPUsername)
	
	// 設定收件者
	m.SetHeader("To", config.MailTo)
	
	// 設定郵件主題
	m.SetHeader("Subject", "IPMAC 系統郵件設定測試")
	
	// 設定郵件內容
	body := fmt.Sprintf(`
<html>
<body>
<h2>IPMAC 系統郵件設定測試</h2>
<p>這是一封來自 IPMAC 系統的測試郵件。</p>
<p><strong>測試時間：</strong> %s</p>
<p><strong>SMTP 伺服器：</strong> %s:%d</p>
<p>如果您收到這封郵件，表示郵件設定正確！</p>
<hr>
<p><small>此郵件由 IPMAC 網路監控系統自動發送</small></p>
</body>
</html>
	`, time.Now().Format("2006-01-02 15:04:05"), config.SMTPServer, config.SMTPPort)
	
	m.SetBody("text/html", body)
	
	// 創建SMTP撥號器
	d := gomail.NewDialer(config.SMTPServer, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
	
	// 處理不同的SMTP設定
	if config.SMTPPort == 465 {
		// SSL/TLS 連接
		d.SSL = true
	} else if config.SMTPPort == 587 || config.SMTPPort == 25 {
		// STARTTLS 連接
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	// gomail.v2 不支援直接設定超時，將使用默認超時
	
	// 發送郵件
	return d.DialAndSend(m)
}

// TestSMTPConnection 測試SMTP連接（不發送郵件）
func TestSMTPConnection(config MailConfig) error {
	d := gomail.NewDialer(config.SMTPServer, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
	
	// 處理不同的SMTP設定
	if config.SMTPPort == 465 {
		// SSL/TLS 連接
		d.SSL = true
	} else if config.SMTPPort == 587 || config.SMTPPort == 25 {
		// STARTTLS 連接
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	// gomail.v2 不支援直接設定超時，將使用默認超時
	
	// 僅測試連接，不發送郵件
	closer, err := d.Dial()
	if err != nil {
		return fmt.Errorf("SMTP 連接失敗: %v", err)
	}
	defer closer.Close()
	
	return nil
}

// SendNotificationMail 發送通知郵件（供後續功能使用）
func SendNotificationMail(config MailConfig, subject, body string) error {
	m := gomail.NewMessage()
	
	// 設定發送者
	m.SetHeader("From", config.SMTPUsername)
	
	// 設定收件者
	m.SetHeader("To", config.MailTo)
	
	// 設定郵件主題
	m.SetHeader("Subject", subject)
	
	// 設定郵件內容
	m.SetBody("text/html", body)
	
	// 創建SMTP撥號器
	d := gomail.NewDialer(config.SMTPServer, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
	
	// 處理不同的SMTP設定
	if config.SMTPPort == 465 {
		// SSL/TLS 連接
		d.SSL = true
	} else if config.SMTPPort == 587 || config.SMTPPort == 25 {
		// STARTTLS 連接
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	// gomail.v2 不支援直接設定超時，將使用默認超時
	
	// 發送郵件
	return d.DialAndSend(m)
}