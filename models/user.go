package models

import "time"

// User struct นี้แมพกับตาราง "users" ใน MySQL db66011212083
type User struct {
    // ใช้ `gorm:"column:..."` เพื่อระบุชื่อ field ใน DB ให้ตรงเป๊ะ
    UserID    uint      `gorm:"primaryKey;autoIncrement;column:user_id" json:"user_id"`
    Username  string    `gorm:"column:username;not null" json:"username"`
    Password  string    `gorm:"column:password;not null" json:"password"`
    Phone     string    `gorm:"column:phone;unique;not null;size:10" json:"phone"`
    Email     string    `gorm:"unique;not null"` 
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}


// ฟังก์ชันนี้บอก GORM ว่า struct นี้คู่กับตารางชื่อ "users"
func (User) TableName() string {
    return "users"
}


// RegisterInput ใช้สำหรับรับค่า JSON จากหน้าบ้านตอนสมัครสมาชิก
type RegisterInput struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
    Phone    string `json:"phone" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
}

type LoginInput struct {
	// รับค่ามาเป็น "username" แต่เราจะเอาไปเช็คว่าเป็น Email หรือ เบอร์โทร ใน Controller
	Username string `json:"username" binding:"required"` 
	Password string `json:"password" binding:"required"`
}

