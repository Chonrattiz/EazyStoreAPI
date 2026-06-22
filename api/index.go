package handler

import (
	"EazyStoreAPI/database"
	_ "EazyStoreAPI/docs"
	"EazyStoreAPI/routes"
	"log"
	"net/http"
	"sync"

	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	handler http.Handler
	once    sync.Once
)

// Handler เป็น Entry Point ที่ Vercel จะเรียกใช้งาน
// Vercel ต้องการ function นี้ใน package handler
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		// โหลด .env (ถ้ามี) - บน Vercel ใช้ Environment Variables แทน
		err := godotenv.Load()
		if err != nil {
			log.Println("Note: .env file not found, using system environment variables instead.")
		}

		// เชื่อมต่อฐานข้อมูล
		database.SetupDatabaseConnection()

		// ตั้งค่า Router
		ginEngine := routes.SetupRouter()

		// เพิ่ม Route สำหรับ Swagger
		ginEngine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		handler = ginEngine
	})

	handler.ServeHTTP(w, r)
}
