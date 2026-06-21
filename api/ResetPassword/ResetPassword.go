package controllers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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

// ✅ SendEmailOTP เปลี่ยนมาใช้ Gmail SMTP
func SendEmailOTP(targetEmail string, otpCode string) error {
	// ดึงรหัสผ่าน 16 หลักจาก Environment Variable
	password := os.Getenv("GMAIL_APP_PASSWORD")

	// ใช้อีเมลทางการของร้านเป็นตัวส่ง
	from := "eazystorepos.official@gmail.com"

	// ตั้งค่า Server ของ Gmail
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// โครงสร้างอีเมล (ใส่ Header ของอีเมลให้ถูกต้อง)
	subject := "Subject: Eazy Store - ยืนยันรหัสผ่านใหม่\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	// โครงสร้าง HTML เดิมที่สวยงาม
	htmlContent := fmt.Sprintf(`
	<html>
	<body style="font-family: Arial, sans-serif;">
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 10px;">
			<h2 style="color: #007bff; text-align: center;">Eazy Store POS</h2>
			<hr>
			<div style="padding: 20px; text-align: center;">
				<p>รหัสยืนยัน (OTP) ของคุณคือ:</p>
				<h1 style="background: #f4f4f4; padding: 15px; display: inline-block; letter-spacing: 5px; color: #333; border-radius: 5px;">%s</h1>
				<p>รหัสนี้จะหมดอายุภายใน <b>10 นาที</b></p>
				<p style="color: #888; font-size: 12px;">หากคุณไม่ได้ขอรหัสนี้ โปรดแจ้งให้เราทราบทันที</p>
			</div>
		</div>
	</body>
	</html>`, otpCode)

	// นำส่วนประกอบทั้งหมดมารวมกันเป็นก้อนเดียว
	msg := []byte(subject + mime + htmlContent)

	// สร้างการยืนยันตัวตนสำหรับล็อกอินเข้า Gmail
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// สั่งส่งอีเมลผ่านพอร์ต 587
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{targetEmail}, msg)
	if err != nil {
		fmt.Println("❌ Error sending email via Gmail SMTP:", err)
		return err
	}

	fmt.Println("✅ ส่ง OTP สำเร็จผ่าน Gmail SMTP!")
	return nil
}

// RequestResetOTP ฟังก์ชันสำหรับรับเรื่องกู้รหัสผ่าน
func RequestResetOTP(c *gin.Context) {
	var input models.ResetRequestInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "กรุณากรอกอีเมลให้ถูกต้อง"})
		return
	}

	// 1. ตรวจสอบว่ามี User นี้ในตาราง users หรือไม่
	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// เพื่อความปลอดภัย ไม่บอกว่าไม่เจออีเมล
		c.JSON(200, gin.H{"message": "หากอีเมลถูกต้อง ระบบจะส่งรหัสไปให้"})
		return
	}

	// 2. เตรียมข้อมูล OTP ใหม่
	otp := GenerateOTP()
	expiresAt := time.Now().Add(10 * time.Minute)

	// 3. ใช้เทคนิค "Upsert" (Update หรือ Insert)
	resetData := models.PasswordReset{
		Email:     input.Email,
		OTPCode:   otp,
		ExpiresAt: expiresAt,
	}

	if err := database.DB.Save(&resetData).Error; err != nil {
		fmt.Println("Database Error:", err)
		c.JSON(500, gin.H{"error": "ไม่สามารถบันทึกข้อมูลได้"})
		return
	}

	// 4. ส่งเมลเบื้องหลังด้วย Goroutine
	go func() {
		err := SendEmailOTP(input.Email, otp)
		if err != nil {
			fmt.Printf("Error sending email to %s: %v\n", input.Email, err)
		}
	}()

	c.JSON(200, gin.H{"message": "ส่งรหัส OTP เรียบร้อยแล้ว"})
}

// VerifyOTP ตรวจสอบรหัสที่ผู้ใช้กรอกมา
func VerifyOTP(c *gin.Context) {
	var input models.VerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	var resetRecord models.PasswordReset
	// ค้นหารหัสจากฐานข้อมูล
	if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTPCode).First(&resetRecord).Error; err != nil {
		c.JSON(401, gin.H{"error": "รหัส OTP ไม่ถูกต้อง"})
		return
	}

	// ตรวจสอบว่าหมดอายุหรือยัง
	if time.Now().After(resetRecord.ExpiresAt) {
		c.JSON(401, gin.H{"error": "รหัส OTP หมดอายุแล้ว"})
		return
	}

	c.JSON(200, gin.H{"message": "ยืนยันรหัส OTP สำเร็จ", "status": "verified"})
}

func UpdatePassword(c *gin.Context) {
	var input models.UpdatePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "กรุณากรอกข้อมูลให้ครบถ้วน"})
		return
	}

	// 1. ตรวจสอบ OTP อีกรอบเพื่อป้องกันการยิง API ข้ามขั้นตอน
	var resetRecord models.PasswordReset
	if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTPCode).First(&resetRecord).Error; err != nil {
		c.JSON(401, gin.H{"error": "ไม่ได้รับอนุญาตให้เปลี่ยนรหัสผ่าน"})
		return
	}

	// 2. แฮชรหัสผ่านใหม่ (bcrypt) เหมือนตอน Register
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 14)

	// 3. อัปเดตในตาราง users
	if err := database.DB.Model(&models.User{}).Where("email = ?", input.Email).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(500, gin.H{"error": "ไม่สามารถเปลี่ยนรหัสผ่านได้"})
		return
	}

	// 4. ลบรหัส OTP ทิ้งทันทีเมื่อใช้เสร็จแล้ว (One-time use)
	database.DB.Delete(&resetRecord)

	c.JSON(200, gin.H{"message": "เปลี่ยนรหัสผ่านสำเร็จแล้ว!"})
}
