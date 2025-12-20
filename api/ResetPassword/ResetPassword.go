package controllers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"time"

	"github.com/gin-gonic/gin"

	// 👇 แก้ Path ให้ตรงกับโฟลเดอร์ในเครื่องของคุณ
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
)

// GenerateOTP ทำหน้าที่สุ่มตัวเลข 6 หลัก
func GenerateOTP() string {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n)
}

// sendEmailOTP ส่ง SMTP ผ่าน Gmail
func sendEmailOTP(targetEmail string, otpCode string) error {
	from := "eazystorepos.official@gmail.com"      // อีเมลร้าน Eazy Store
	password := "bxow wqtp lgks ahnn"            // App Password 16 หลักที่คุณขอมา

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	subject := "Subject: Eazy Store - รหัสยืนยันการเปลี่ยนรหัสผ่าน\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	body := "รหัส OTP ของคุณคือ: " + otpCode + "\nกรุณาใช้งานภายใน 10 นาทีเพื่อความปลอดภัย"
	message := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{targetEmail}, message)
}

// RequestResetOTP ฟังก์ชันหลักสำหรับรับเรื่องกู้รหัสผ่าน
func RequestResetOTP(c *gin.Context) {
	var input models.ResetRequestInput // เรียกใช้ Input จาก models

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "กรุณากรอกอีเมลให้ถูกต้อง"})
		return
	}

	// 🔥 1. ตรวจสอบว่ามี User นี้ในตาราง users หรือไม่ (ใช้ database.DB)
	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// เพื่อความปลอดภัย ไม่บอกว่าไม่เจออีเมล
		c.JSON(200, gin.H{"message": "หากอีเมลถูกต้อง ระบบจะส่งรหัสไปให้"})
		return
	}

	// 2. สุ่มรหัส OTP
	otp := GenerateOTP()

	// 🔥 3. บันทึกลงตาราง PasswordReset โดยใช้ database.DB
	resetData := models.PasswordReset{
		Email:     input.Email,
		OTPCode:   otp,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	
	// บันทึก/อัปเดตข้อมูลลงฐานข้อมูล (ถ้าซ้ำจะ Update ทับตัวเก่าให้เอง)
	if err := database.DB.Save(&resetData).Error; err != nil {
		c.JSON(500, gin.H{"error": "ไม่สามารถบันทึกข้อมูล OTP ได้"})
		return
	}

	// 4. ส่งเมลเบื้องหลังด้วย Goroutine (เพื่อไม่ให้ API ตอบกลับช้า)
	go func() {
		err := sendEmailOTP(input.Email, otp)
		if err != nil {
			fmt.Printf("Error sending email to %s: %v\n", input.Email, err)
		}
	}()

	c.JSON(200, gin.H{"message": "ส่งรหัส OTP เรียบร้อยแล้ว"})
}