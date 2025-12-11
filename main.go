// File: main.go
package main

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/routes"
)

func main() {
	// 1. เริ่มต้นเชื่อมต่อ Database
	database.SetupDatabaseConnection()

	// 2. ตั้งค่า Router
	r := routes.SetupRouter()

	// 3. รัน Server ที่ Port 8080
	r.Run(":8080")
}