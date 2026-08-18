package controllers

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	// 👇 แก้ Path ให้ตรงกับโฟลเดอร์ในเครื่องของคุณ
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
)

func GenerateOTP() string {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n)
}

func SendEmailOTP(targetEmail string, otpCode string, subject string) error {

	gasURL := "https://script.google.com/macros/s/AKfycbx68CqYnpCakrLJ3KmMHHKJxlKbuRzFqqcseE3K9A-NOGMjhVYUCTpJo6p5Kq0UHzvw/exec"

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

	// สร้างข้อมูล JSON สำหรับส่งไปให้ Google
	requestBody, _ := json.Marshal(map[string]string{
		"to":       targetEmail,
		"subject":  subject,
		"htmlBody": htmlContent,
	})

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Post(gasURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("❌ Error connecting to Google Script:", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		fmt.Println("✅ ส่ง OTP สำเร็จผ่าน Google Apps Script ทะลุ Render แล้ว!")
		return nil
	}

	return fmt.Errorf("failed to send email, status code: %d", resp.StatusCode)
}

func RequestResetOTP(c *gin.Context) {
	var input models.ResetRequestInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "กรุณากรอกอีเมลให้ถูกต้อง"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(200, gin.H{"message": "หากอีเมลถูกต้อง ระบบจะส่งรหัสไปให้"})
		return
	}


	otp := GenerateOTP()
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute)


	resetData := models.PasswordReset{
		Email:     input.Email,
		OTPCode:   otp,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := database.DB.Save(&resetData).Error; err != nil {
		fmt.Println("Database Error:", err)
		c.JSON(500, gin.H{"error": "ไม่สามารถบันทึกข้อมูลได้"})
		return
	}

	if err := SendEmailOTP(input.Email, otp, "Eazy Store - ยืนยันรหัสผ่านใหม่"); err != nil {
		fmt.Printf("Error sending email to %s: %v\n", input.Email, err)
		c.JSON(500, gin.H{"error": "ไม่สามารถส่งอีเมล OTP ได้ กรุณาลองใหม่อีกครั้ง"})
		return
	}

	c.JSON(200, gin.H{"message": "ส่งรหัส OTP เรียบร้อยแล้ว"})
}


func VerifyOTP(c *gin.Context) {
	var input models.VerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	var resetRecord models.PasswordReset

	if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTPCode).First(&resetRecord).Error; err != nil {
		c.JSON(401, gin.H{"error": "รหัส OTP ไม่ถูกต้อง"})
		return
	}


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

	
	var resetRecord models.PasswordReset
	if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTPCode).First(&resetRecord).Error; err != nil {
		c.JSON(401, gin.H{"error": "ไม่ได้รับอนุญาตให้เปลี่ยนรหัสผ่าน"})
		return
	}


	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 14)


	if err := database.DB.Model(&models.User{}).Where("email = ?", input.Email).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(500, gin.H{"error": "ไม่สามารถเปลี่ยนรหัสผ่านได้"})
		return
	}

	
	database.DB.Delete(&resetRecord)

	c.JSON(200, gin.H{"message": "เปลี่ยนรหัสผ่านสำเร็จแล้ว!"})
}
