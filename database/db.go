// File: database/db.go
package database

import (
	models "EazyStoreAPI/models"
	"fmt"
	"time"

	"gorm.io/driver/mysql" // เปลี่ยนจาก sqlite เป็น mysql
	"gorm.io/gorm"
)

var DB *gorm.DB

func SetupDatabaseConnection() {

	dsn := "66011212083:66011212083@tcp(202.28.34.210:3309)/db66011212083?charset=utf8mb4&parseTime=True&loc=Local"

	// เซิร์ฟเวอร์ (Render) ตั้ง OS timezone เป็น UTC ค่าเริ่มต้น
	// ถ้าไม่แปลงก่อน created_at/updated_at ที่ GORM เติมอัตโนมัติจะเพี้ยนไป 7 ชั่วโมง
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			loc, err := time.LoadLocation("Asia/Bangkok")
			if err != nil {
				loc = time.FixedZone("ICT", 7*60*60)
			}
			return time.Now().In(loc)
		},
	})

	if err != nil {
		panic("เชื่อมต่อฐานข้อมูลไม่สำเร็จ ❌: " + err.Error())
	}

	// เชื่อมต่อสำเร็จ
	fmt.Println("เชื่อมต่อฐานข้อมูล db66011212083 สำเร็จแล้ว! ✅")

	// เซิร์ฟเวอร์ MySQL ปิด connection ที่ idle นานเกิน wait_timeout ของมันเองแบบเงียบๆ
	// ถ้าไม่ตั้งอายุ connection ในฝั่ง Go ไว้ พอ connection ใน pool ค้างว่างนานเกินไป
	// ครั้งถัดไปที่หยิบมาใช้จะเจอ error "invalid connection" เพราะปลายทางตัดสายไปแล้ว
	sqlDB, err := database.DB()
	if err != nil {
		panic("ไม่สามารถตั้งค่า connection pool ได้ ❌: " + err.Error())
	}
	sqlDB.SetConnMaxLifetime(3 * time.Minute)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)

	// AutoMigrate: เช็คว่า Struct ใน Go ตรงกับ Table ใน MySQL ไหม
	// ถ้ายังไม่มีตาราง users ระบบจะสร้างให้ (แต่เราสร้างไว้แล้ว มันจะแค่เช็คเฉยๆ)
	database.AutoMigrate(&models.User{})
	database.AutoMigrate(&models.RefreshToken{})
	database.AutoMigrate(&models.PasswordReset{})
	database.AutoMigrate(&models.EmailVerification{})

	DB = database
}
