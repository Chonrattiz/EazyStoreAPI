package models

import (
	"time"
)

// PasswordReset สำหรับเก็บรหัส OTP ในฐานข้อมูล
type PasswordReset struct {
    Email     string    `gorm:"primaryKey;type:varchar(255)" json:"email"`
    OTPCode   string    `gorm:"not null" json:"otp_code"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

// โครงสร้างรับค่าสำหรับ API
type ResetRequestInput struct {
    Email string `json:"email" binding:"required,email"`
}