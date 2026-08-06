package models

import (
	"time"
)

// PasswordReset สำหรับเก็บรหัส OTP ในฐานข้อมูล
type PasswordReset struct {
    Email     string    `gorm:"primaryKey;type:varchar(100)" json:"email"`
    OTPCode   string    `gorm:"not null" json:"otp_code"`
    // ไม่มี <-:create เพราะอยากให้ created_at รีเฟรชเป็นเวลาที่ขอ OTP รอบล่าสุดทุกครั้งที่กดขอใหม่ (resend)
    // precision:0 กัน GORM เติมทศนิยมวินาที (datetime(3)) ให้อัตโนมัติ จะได้ไม่มี .513 ต่อท้าย
    CreatedAt time.Time `gorm:"precision:0;not null" json:"created_at"`
    ExpiresAt time.Time `gorm:"precision:0;not null" json:"expires_at"`
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