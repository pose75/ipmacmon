package main

import (
	"github.com/gofiber/fiber/v2"
)

func setupRoutes(app *fiber.App) {
	// 首頁
	app.Get("/", indexHandler)
	
	// API 路由群組
	api := app.Group("/api")
	
	// ARP 相關 API
	api.Get("/arp-entries", getArpEntriesHandler)
	api.Get("/scan-batches", getScanBatchesHandler)
	api.Post("/scan", scanHandler)
	
	// 網路設定 API
	api.Get("/config", getNetworkConfigHandler)
	api.Post("/config", updateNetworkConfigHandler)
	
	// 自動掃描設定 API
	api.Get("/auto-scan-config", getAutoScanConfigHandler)
	api.Post("/auto-scan-config", updateAutoScanConfigHandler)
	
	// 郵件設定 API
	api.Get("/mail-config", getMailConfigHandler)
	api.Post("/mail-config", updateMailConfigHandler)
	api.Post("/test-mail", testMailHandler)
	
	// 自動掃描控制 API
	api.Post("/auto-scan-control", autoScanControlHandler)
	api.Get("/auto-scan-status", autoScanStatusHandler)
	
	// IP/MAC 對應維護 API
	api.Get("/ip-mac-mappings", getIPMacMappingsHandler)
	api.Post("/ip-mac-mappings", updateIPMacMappingHandler)
	api.Delete("/ip-mac-mappings/clear-all", clearAllIPMacMappingsHandler)
	api.Delete("/ip-mac-mappings/:id", deleteIPMacMappingHandler)
	api.Get("/check-ip/:ip", checkIPExistsHandler)
}