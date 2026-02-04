// File: main.go
package main

import (
	"EazyStoreAPI/database"
	_ "EazyStoreAPI/docs" // import swagger docs
	"EazyStoreAPI/routes" // <--- เพิ่ม import นี้
	"log"

	"github.com/joho/godotenv" // <--- เพิ่ม import นี้

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
	// 1. โหลดไฟล์ .env ก่อนเริ่มทำงาน
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file") // ถ้าหาไฟล์ไม่เจอให้แจ้งเตือน
	}
	// 1. เชื่อมต่อ Database
	database.SetupDatabaseConnection()

	// 2. ตั้งค่า Router
	r := routes.SetupRouter()

	// 3. เพิ่ม Route สำหรับ Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 4. รัน Server
	r.Run(":8080")
}
