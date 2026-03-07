package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	resetController "EazyStoreAPI/api/ResetPassword"
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
)

// UpdateProfile สำหรับแก้ไขข้อมูลส่วนตัว
// @Summary      แก้ไขโปรไฟล์ผู้ใช้
// @Description  แก้ไขข้อมูลส่วนตัว (หากเปลี่ยนอีเมลต้องยืนยันตัวตนใหม่)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Router       /api/profile/update [put]
func UpdateProfile(c *gin.Context) {
	// 1. รับ UserID จาก JWT Token (สมมติว่า Auth Middleware เซ็ตค่า "user_id" ไว้ให้)
	// หมายเหตุ: JWT มักจะแปลงตัวเลขเป็น float64 เราเลยต้อง cast ให้ถูกต้อง
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลยืนยันตัวตน (กรุณา Login)"})
		return
	}
	
	// แปลง type ให้เป็น uint ตามโมเดล
	var userID uint
	switch v := userIDValue.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "รูปแบบ UserID ไม่ถูกต้อง"})
		return
	}

	// 2. ค้นหาข้อมูล User เดิมจาก DB
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบบัญชีผู้ใช้งานในระบบ"})
		return
	}

	// 3. รับข้อมูลที่ส่งมาแบบ Map (เพื่อทำ Partial Update - อัปเดตเฉพาะค่าที่ส่งมา)
	var inputMap map[string]interface{}
	if err := c.ShouldBindJSON(&inputMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	updateData := make(map[string]interface{})
	needsOTP := false // ตัวแปรเช็คว่าต้องส่ง OTP ยืนยันอีเมลไหม
	var newEmail string

	// 4. กรองและตรวจสอบข้อมูลที่จะแก้ไข
	// 4.1 แก้ไข Username
	if val, ok := inputMap["username"]; ok {
		newUsername := val.(string)
		if newUsername != "" && newUsername != user.Username {
			// เช็คว่า Username ซ้ำไหม
			var count int64
			database.DB.Model(&models.User{}).Where("username = ?", newUsername).Count(&count)
			if count > 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "ชื่อผู้ใช้งาน (Username) นี้มีคนใช้แล้ว"})
				return
			}
			updateData["username"] = newUsername
		}
	}

	// 4.2 แก้ไข Password (ต้องเข้ารหัสใหม่)
	if val, ok := inputMap["password"]; ok {
		passwordStr := val.(string)
		if passwordStr != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordStr), 14) // ใช้ Cost 14 ตามโค้ดเดิมของคุณ
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถเข้ารหัสผ่านได้"})
				return
			}
			updateData["password"] = string(hashedPassword)
		}
	}

	// 4.3 แก้ไข Phone
	if val, ok := inputMap["phone"]; ok {
		newPhone := val.(string)
		if newPhone != "" && newPhone != user.Phone {
			var count int64
			database.DB.Model(&models.User{}).Where("phone = ?", newPhone).Count(&count)
			if count > 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "เบอร์โทรศัพท์นี้ถูกใช้งานแล้ว"})
				return
			}
			updateData["phone"] = newPhone
		}
	}

	// 4.4 ✨ แก้ไข Email (ต้องบังคับยืนยันตัวตนใหม่)
	if val, ok := inputMap["email"]; ok {
		newEmail = val.(string)
		
		// ถ้ามีการส่งอีเมลมา และไม่ใช่อีเมลเดิม
		if newEmail != "" && newEmail != user.Email {
			var count int64
			database.DB.Model(&models.User{}).Where("email = ?", newEmail).Count(&count)
			if count > 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "อีเมลนี้มีผู้ใช้งานในระบบแล้ว"})
				return
			}

			// เปลี่ยนค่าใน Map และบังคับ IsVerified กลับไปเป็น false ทันที
			updateData["email"] = newEmail
			updateData["is_verified"] = false
			needsOTP = true
		}
	}

	// 5. สั่งอัปเดตลง Database
	if len(updateData) > 0 {
		if err := database.DB.Model(&user).Updates(updateData).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "อัปเดตข้อมูลไม่สำเร็จ: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":      "ไม่มีข้อมูลเปลี่ยนแปลง",
			"user":         user,
			"require_auth": false,
		})
		return
	}

	// 6. ✨ สร้าง OTP และส่ง Email (ใช้ฟังก์ชันเดิมของคุณเลย)
	if needsOTP {
		otp := resetController.GenerateOTP()
		
		// บันทึกลงตาราง EmailVerification
		verification := models.EmailVerification{
			Email:     newEmail,
			OTPCode:   otp,
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}
		database.DB.Save(&verification)

		// สั่งส่งอีเมลแบบ Asynchronous (ไม่บล็อกการทำงาน)
		go resetController.SendEmailOTP(newEmail, otp)
	}

	// 7. ดึงข้อมูลล่าสุดที่อัปเดตเสร็จแล้ว ส่งกลับไปให้หน้าแอป
	var updatedUser models.User
	database.DB.First(&updatedUser, userID)

	c.JSON(http.StatusOK, gin.H{
		"message":      "อัปเดตโปรไฟล์สำเร็จ",
		"user":         gin.H{
			"id":       updatedUser.UserID,
			"username": updatedUser.Username,
			"email":    updatedUser.Email,
			"phone":    updatedUser.Phone,
		},
		"require_auth": needsOTP, // ✨ คืนค่า True เพื่อบอก Flutter ให้เด้งไปหน้ายืนยัน OTP
	})
}

// GetProfile ดึงข้อมูลโปรไฟล์ของผู้ใช้งานปัจจุบัน
// @Summary      ดึงข้อมูลโปรไฟล์
// @Description  ดึงข้อมูลส่วนตัวของผู้ใช้ที่ Login อยู่
// @Tags         Profile
// @Produce      json
// @Security     BearerAuth
// @Router       /api/profile [get]
func GetProfile(c *gin.Context) {
	// 1. รับ UserID จาก JWT Token (ดึงมาจาก Middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลยืนยันตัวตน (กรุณา Login)"})
		return
	}

	// แปลง type ให้เป็น uint ตามโมเดล
	var userID uint
	switch v := userIDValue.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "รูปแบบ UserID ไม่ถูกต้อง"})
		return
	}

	// 2. ค้นหาข้อมูล User ล่าสุดจาก Database
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบบัญชีผู้ใช้งานในระบบ"})
		return
	}

	// 3. ส่งข้อมูลกลับไปให้หน้าแอป (Flutter) 
	// ⚠️ สำคัญ: เราจะไม่ส่ง Password กลับไปเด็ดขาดเพื่อความปลอดภัย
	c.JSON(http.StatusOK, gin.H{
		"message": "ดึงข้อมูลโปรไฟล์สำเร็จ",
		"user": gin.H{
			"id":          user.UserID,
			"username":    user.Username,
			"email":       user.Email,
			"phone":       user.Phone,
			"is_verified": user.IsVerified, // ส่งสถานะยืนยันอีเมลกลับไปด้วย
			"created_at":  user.CreatedAt,
		},
	})
}