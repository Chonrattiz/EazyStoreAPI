package models

import (
	"time"
)

// PasswordReset สำหรับเก็บรหัส OTP ในฐานข้อมูล
type PasswordReset struct {
    Email     string    `gorm:"primaryKey;type:varchar(100)" json:"email"`
    OTPCode   string    `gorm:"not null" json:"otp_code"`
    // เพิ่ม <-:create เพื่อบอกว่าให้เขียนค่าเฉพาะตอนสร้าง (Create) เท่านั้น
    CreatedAt time.Time `gorm:"autoCreateTime;<-:create" json:"created_at"` 
    ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

// --- โซน Input สำหรับรับค่าจาก API (DTO) ---

// ResetRequestInput สำหรับขั้นตอนขอ OTP
type ResetRequestInput struct {
    Email string `json:"email" binding:"required,email"`
}

// VerifyOTPInput สำหรับขั้นตอนตรวจสอบรหัส OTP
type VerifyOTPInput struct {
    Email   string `json:"email" binding:"required,email"`
    OTPCode string `json:"otp_code" binding:"required"`
}

// UpdatePasswordInput สำหรับขั้นตอนตั้งรหัสผ่านใหม่
type UpdatePasswordInput struct {
    Email       string `json:"email" binding:"required,email"`
    NewPassword string `json:"new_password" binding:"required"`
    OTPCode     string `json:"otp_code" binding:"required"`
}