package models

import "time"

// User struct นี้แมพกับตาราง "users" ใน MySQL db66011212083
type User struct {
	// ใช้ `gorm:"column:..."` เพื่อระบุชื่อ field ใน DB ให้ตรงเป๊ะ
	UserID     uint      `gorm:"primaryKey;autoIncrement;column:user_id" json:"user_id"`
	Username   string    `gorm:"column:username;not null" json:"username"`
	Password   string    `gorm:"column:password;not null" json:"password"`
	Phone      string    `gorm:"column:phone;unique;not null;size:10" json:"phone"`
	Email      string    `gorm:"unique;not null"`
	ShopID     int       `gorm:"column:shop_id;not null" json:"shop_id"` // ✨ ผูกกับร้านค้า
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	IsVerified bool      `gorm:"default:false" json:"is_verified"` // ✨ เพิ่มฟิลด์นี้
}

// โครงสร้างสำหรับยืนยันอีเมลตอนสมัคร
type EmailVerification struct {
	Email     string    `gorm:"primaryKey;type:varchar(100)" json:"email"`
	OTPCode   string    `gorm:"not null" json:"otp_code"`
	// ไม่มี <-:create เพราะอยากให้ created_at รีเฟรชเป็นเวลาที่ขอ OTP รอบล่าสุดทุกครั้งที่กดขอใหม่ (resend)
	// precision:0 กัน GORM เติมทศนิยมวินาที (datetime(3)) ให้อัตโนมัติ จะได้ไม่มี .513 ต่อท้าย
	CreatedAt time.Time `gorm:"precision:0;not null" json:"created_at"`
	ExpiresAt time.Time `gorm:"precision:0;not null" json:"expires_at"`
}

// ฟังก์ชันนี้บอก GORM ว่า struct นี้คู่กับตารางชื่อ "users"
func (User) TableName() string {
	return "users"
}

// RegisterInput ใช้สำหรับรับค่า JSON จากหน้าบ้านตอนสมัครสมาชิก
type RegisterInput struct {
	Username string `json:"username" binding:"required" example:"poder"`
	Password string `json:"password" binding:"required" example:"123456"`
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type LoginInput struct {
	// รับค่ามาเป็น "username" แต่เราจะเอาไปเช็คว่าเป็น Email หรือ เบอร์โทร ใน Controller
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// DeviceID ไม่บังคับ ใช้แยกว่า refresh token นี้เป็นของเครื่องไหน เพื่อให้ logout เฉพาะเครื่องได้
	DeviceID string `json:"device_id"`
	// DeviceName ไม่บังคับ ชื่อรุ่นเครื่องแบบอ่านง่าย เก็บไว้โชว์ผลเฉยๆ
	DeviceName string `json:"device_name"`
}
