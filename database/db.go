// File: database/db.go
package database

import (
	models "EazyStoreAPI/models"
	"fmt"

	"gorm.io/driver/mysql" // เปลี่ยนจาก sqlite เป็น mysql
	"gorm.io/gorm"
)

var DB *gorm.DB


func SetupDatabaseConnection() {
   
  dsn := "66011212083:66011212083@tcp(202.28.34.210:3309)/db66011212083?charset=utf8mb4&parseTime=True&loc=Local"

    database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

    if err != nil {
        panic("เชื่อมต่อฐานข้อมูลไม่สำเร็จ ❌: " + err.Error())
    }

    // เชื่อมต่อสำเร็จ
    fmt.Println("เชื่อมต่อฐานข้อมูล db66011212083 สำเร็จแล้ว! ✅")

    // AutoMigrate: เช็คว่า Struct ใน Go ตรงกับ Table ใน MySQL ไหม
    // ถ้ายังไม่มีตาราง users ระบบจะสร้างให้ (แต่เราสร้างไว้แล้ว มันจะแค่เช็คเฉยๆ)
    database.AutoMigrate(&models.User{})

    DB = database
}


