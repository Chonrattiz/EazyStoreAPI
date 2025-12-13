// File: main.go
package main

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/routes"

	_ "EazyStoreAPI/docs" // ⭐ สำคัญมาก: import swagger docs

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           EazyStore API
// @version         1.0
// @description     API สำหรับระบบ EazyStore
// @host            localhost:8080
// @BasePath        /
func main() {
	// 1. เชื่อมต่อ Database
	database.SetupDatabaseConnection()

	// 2. ตั้งค่า Router
	r := routes.SetupRouter()

	// 3. เพิ่ม Route สำหรับ Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 4. รัน Server
	r.Run(":8080")
}
