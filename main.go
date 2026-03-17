package main

import (
	"log"

	"git-file-transfer/internal/api"
	"git-file-transfer/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 创建文件管理器
	uploadDir := "./uploads"
	fileManager := storage.NewFileManager(uploadDir)

	log.Printf("Starting file transfer server...")
	log.Printf("Upload directory: %s", uploadDir)
	log.Printf("Max file size: %d MB", storage.MaxFileSize/(1024*1024))
	log.Printf("Default chunk size: %d MB", storage.DefaultChunkSize/(1024*1024))

	// 设置路由
	api.SetupRoutes(r, fileManager)

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "file-transfer",
		})
	})

	// 启动服务器
	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
