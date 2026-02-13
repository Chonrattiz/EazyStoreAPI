package main

import (
	"EazyStoreAPI/database"
	_ "EazyStoreAPI/docs"
	"EazyStoreAPI/routes"
	"log"
	"os" // <--- อย่าลืมเพิ่ม import os

	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           EazyStore API
// @version         1.0
// @description     API สำหรับระบบ EazyStore
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
    // 1. โหลดไฟล์ .env (ถ้ามี)
    err := godotenv.Load()
    if err != nil {
        // แก้: เปลี่ยนจาก Fatal เป็น Println เพราะบน Cloud ไม่มีไฟล์ .env ก็รันได้ (ใช้ Env Var ของระบบแทน)
        log.Println("Note: .env file not found, using system environment variables instead.")
    }

    // 2. เชื่อมต่อ Database
    database.SetupDatabaseConnection()

    // 3. ตั้งค่า Router
    r := routes.SetupRouter()

    // 4. เพิ่ม Route สำหรับ Swagger
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // 5. รัน Server (แก้เรื่อง Port)
    port := os.Getenv("PORT") // อ่าน Port จาก Render
    if port == "" {
        port = "8080" // ถ้าไม่มี (เช่นรันในเครื่องตัวเอง) ให้ใช้ 8080
    }

    // ใช้ 0.0.0.0 เพื่อให้เข้าถึงได้จากภายนอก Container
    r.Run("0.0.0.0:" + port) 
}
