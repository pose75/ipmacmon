package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
)

//go:embed templates
var templateFS embed.FS




func main() {
	// 初始化資料庫
	initDatabase()

	// 初始化模板引擎（使用嵌入的檔案系統）
	// 需要從 embed.FS 中取得子目錄
	templatesSubFS, err := fs.Sub(templateFS, "templates")
	if err != nil {
		log.Fatal("無法取得 templates 子目錄:", err)
	}
	
	// 將 fs.FS 轉換為 http.FileSystem
	httpFS := http.FS(templatesSubFS)
	engine := html.NewFileSystem(httpFS, ".html")

	// 建立 Fiber 應用
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 中間件
	app.Use(logger.New())
	app.Use(cors.New())

	// 路由
	setupRoutes(app)

	// 啟動服務器
	log.Println("服務器啟動在 http://localhost:9999")
	log.Fatal(app.Listen(":9999"))
}
