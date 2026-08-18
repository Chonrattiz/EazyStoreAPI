package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt" // <--- Import ตัวนี้เพิ่มมา

	// 👇 แก้ Path ให้ตรงกับโฟลเดอร์ในเครื่องของคุณ
	resetController "EazyStoreAPI/controllers/ResetPassword"
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
)

// nowInBangkok คืนเวลาปัจจุบันแบบเวลาไทย เพราะเซิร์ฟเวอร์ (Render) ใช้ OS timezone เป็น UTC
func nowInBangkok() time.Time {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	return time.Now().In(loc)
}

// --------------------------------------------------------------------
// ฟังก์ชัน Register
// --------------------------------------------------------------------
// @Summary      สมัครสมาชิก
// @Description  ลงทะเบียนผู้ใช้ใหม่เข้าสู่ระบบ
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        input body models.RegisterInput true "ข้อมูลสำหรับสมัครสมาชิก"
// @Success      200 {object} object "status: success"
// @Router       /api/auth/register [post]
func Register(c *gin.Context) {
	var input models.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลสมัครสมาชิกไม่ครบถ้วน กรุณากรอกให้ครบทุกช่อง"})
		return
	}

	var existingUser models.User
	result := database.DB.Where("email = ? OR phone = ?", input.Email, input.Phone).First(&existingUser)

	if result.Error == nil {
		// ... (กรณีเคยสมัครค้างไว้)
		if existingUser.IsVerified {
			c.JSON(http.StatusConflict, gin.H{"error": "Email หรือ เบอร์โทรนี้ถูกใช้งานแล้ว"})
			return
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
		existingUser.Username = input.Username
		existingUser.Email = input.Email
		existingUser.Phone = input.Phone
		existingUser.Password = string(hashedPassword)
		if err := database.DB.Save(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสมัครสมาชิกได้"})
			return
		}
	} else {
		// ... (กรณีผู้ใช้ใหม่ 100%)
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
		existingUser = models.User{
			Username:   input.Username,
			Password:   string(hashedPassword),
			Email:      input.Email,
			Phone:      input.Phone,
			IsVerified: false,
		}
		if err := database.DB.Create(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างบัญชีได้"})
			return
		}
	}

	otp := resetController.GenerateOTP()
	now := time.Now()
	verification := models.EmailVerification{
		Email:     input.Email,
		OTPCode:   otp,
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := database.DB.Save(&verification).Error; err != nil {
		fmt.Println("Register - Database Error saving EmailVerification:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกรหัส OTP ได้"})
		return
	}

	go func() {
		err := resetController.SendEmailOTP(input.Email, otp, "Eazy Store - ยืนยันรหัส OTP")
		if err != nil {
			fmt.Printf("Register - Error sending email OTP to %s: %v\n", input.Email, err)
		} else {
			fmt.Printf("Register - Successfully sent OTP to %s\n", input.Email)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "ระบบส่งรหัส OTP ไปยังอีเมลใหม่เรียบร้อยแล้ว"})
}

func VerifyRegistration(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required"`
		OTP   string `json:"otp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "กรุณากรอกข้อมูลให้ครบ"})
		return
	}

	var record models.EmailVerification

	if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTP).First(&record).Error; err != nil {
		c.JSON(400, gin.H{"error": "รหัส OTP ไม่ถูกต้อง หรือหมดอายุ"})
		return
	}

	if time.Now().After(record.ExpiresAt) {
		c.JSON(400, gin.H{"error": "รหัส OTP หมดอายุแล้ว"})
		return
	}

	database.DB.Model(&models.User{}).Where("email = ?", input.Email).Update("is_verified", true)

	database.DB.Delete(&record)

	c.JSON(200, gin.H{"message": "ยืนยันอีเมลสำเร็จแล้ว คุณสามารถเข้าสู่ระบบได้ทันที"})
}

func ChangeEmailBeforeVerify(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		NewEmail string `json:"new_email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ? AND is_verified = ?", input.Username, false).First(&user).Error; err != nil {
		c.JSON(404, gin.H{"error": "ไม่พบข้อมูลผู้ใช้ที่รอการยืนยัน"})
		return
	}

	user.Email = input.NewEmail
	database.DB.Save(&user)

	// ส่ง OTP ใหม่ไปที่เมลใหม่
	otp := resetController.GenerateOTP()
	now := time.Now()
	verification := models.EmailVerification{
		Email:     input.NewEmail,
		OTPCode:   otp,
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := database.DB.Save(&verification).Error; err != nil {
		fmt.Println("ChangeEmailBeforeVerify - Database Error saving EmailVerification:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกรหัส OTP ได้"})
		return
	}

	go func() {
		err := resetController.SendEmailOTP(input.NewEmail, otp, "Eazy Store - ยืนยันรหัส OTP")
		if err != nil {
			fmt.Printf("ChangeEmailBeforeVerify - Error sending email OTP to %s: %v\n", input.NewEmail, err)
		} else {
			fmt.Printf("ChangeEmailBeforeVerify - Successfully sent OTP to %s\n", input.NewEmail)
		}
	}()

	c.JSON(200, gin.H{"message": "อัปเดตอีเมลและส่งรหัส OTP ใหม่แล้ว"})
}

// --------------------------------------------------------------------
// ฟังก์ชัน Login (เข้าสู่ระบบ) - รองรับการเช็ค Hash
// --------------------------------------------------------------------
// @Summary      เข้าสู่ระบบ (Login)
// @Description  ล็อกอินด้วย Username, Email หรือ เบอร์โทรศัพท์
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body models.LoginInput true "ข้อมูลสำหรับ Login"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/auth/login [post]
// Login ฟังก์ชันสำหรับการเข้าสู่ระบบ
func Login(c *gin.Context) {
	var input models.LoginInput

	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณากรอกข้อมูลให้ครบถ้วน"})
		return
	}

	var user models.User

	invalidMsg := "อีเมล/เบอร์โทรศัพท์หรือรหัสผ่านไม่ถูกต้อง"


	if err := database.DB.Where("email = ? OR phone = ?", input.Username, input.Username).First(&user).Error; err != nil {
	
		c.JSON(http.StatusUnauthorized, gin.H{"error": invalidMsg})
		return
	}

	
	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "กรุณายืนยันตัวตนผ่าน OTP ก่อนเข้าสู่ระบบ",
			"is_verified": false,
			"email":       user.Email,   
			"username":    user.Username,
		})
		return
	}


	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": invalidMsg})
		return
	}


	accessTokenExpiry := nowInBangkok().Add(15 * time.Minute)
	accessClaims := jwt.MapClaims{
		"user_id":  user.UserID,
		"username": user.Username,
		"type":     "access",
		"exp":      accessTokenExpiry.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	secretKey := os.Getenv("JWT_SECRET")
	accessTokenString, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างรหัสการเข้าถึงได้"})
		return
	}


	refreshTokenExpiry := nowInBangkok().AddDate(0, 1, 0)
	refreshClaims := jwt.MapClaims{
		"user_id": user.UserID,
		"type":    "refresh",
		"exp":     refreshTokenExpiry.Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้าง refresh token ได้"})
		return
	}

	
	refreshTokenRecord := models.RefreshToken{
		UserID:    user.UserID,
		Token:     refreshTokenString,
		ExpiresAt: refreshTokenExpiry,
	}
	if input.DeviceID != "" {
		deviceID := input.DeviceID
		refreshTokenRecord.DeviceID = &deviceID
	}
	if input.DeviceName != "" {
		deviceName := input.DeviceName
		refreshTokenRecord.DeviceName = &deviceName
	}
	if err := database.DB.Create(&refreshTokenRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึก token ได้"})
		return
	}

	
	c.JSON(http.StatusOK, gin.H{
		"message":       "เข้าสู่ระบบสำเร็จ",
		"access_token":  accessTokenString,
		"refresh_token": refreshTokenString,
		"expires_in":    900, // 15 นาที
		"user": gin.H{
			"id":       user.UserID,
			"username": user.Username,
			"email":    user.Email,
			"phone":    user.Phone,
		},
	})
}

// ============================================================
// Refresh Token - ใช้เมื่อ Access Token หมดอายุ
// ============================================================
// @Summary      รีเฟรช Access Token
// @Description  ใช้ Refresh Token เพื่อขอ Access Token ใหม่
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body map[string]string true "Refresh Token"
// @Success      200 {object} map[string]interface{}
// @Router       /api/auth/refresh [post]
func Refresh(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง refresh_token"})
		return
	}

	// 1. ตรวจสอบ Refresh Token ว่า valid หรือไม่
	secretKey := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(input.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token ไม่ถูกต้อง"})
		return
	}

	// 2. ตรวจสอบว่า token type เป็น refresh
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token type ไม่ถูกต้อง"})
		return
	}

	// 3. ดึง user_id จาก claims
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ user_id ใน token"})
		return
	}
	userID := uint(userIDFloat)

	// 4. ตรวจสอบว่า refresh token ยังเก็บไว้ในฐานข้อมูลและยังไม่โดนยกเลิก
	var refreshTokenRecord models.RefreshToken
	result := database.DB.Where("user_id = ? AND token = ? AND revoked_at IS NULL", userID, input.RefreshToken).First(&refreshTokenRecord)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token ไม่พบหรือโดนยกเลิกแล้ว"})
		return
	}

	// 5. ตรวจสอบว่า refresh token ยังไม่หมดอายุ
	if nowInBangkok().After(refreshTokenRecord.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token หมดอายุแล้ว"})
		return
	}

	// 6. ดึงข้อมูล User
	var user models.User
	if err := database.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบผู้ใช้"})
		return
	}

	// 7. สร้าง Access Token ใหม่
	accessTokenExpiry := nowInBangkok().Add(15 * time.Minute)
	accessClaims := jwt.MapClaims{
		"user_id":  user.UserID,
		"username": user.Username,
		"type":     "access",
		"exp":      accessTokenExpiry.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้าง access token ได้"})
		return
	}

	// 8. ส่ง Access Token ใหม่ตอบกลับ
	c.JSON(http.StatusOK, gin.H{
		"message":      "รีเฟรช Access Token สำเร็จ",
		"access_token": accessTokenString,
		"expires_in":   900, // 15 นาที = 900 วินาที

	})
}

// ============================================================
// Logout - ยกเลิก Refresh Token
// ============================================================
// @Summary      ออกจากระบบ (Logout)
// @Description  ยกเลิก Refresh Token ของผู้ใช้
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer Access Token"
// @Success      200 {object} map[string]string
// @Router       /api/auth/logout [post]
func Logout(c *gin.Context) {
	// 0. ดึง device_id จาก body (ไม่บังคับ) — ถ้าส่งมาจะ logout เฉพาะเครื่องนั้น ถ้าไม่ส่งจะ logout ทุกเครื่อง
	var body struct {
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&body)

	// 1. ดึง Access Token จาก header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ไม่พบ Authorization header"})
		return
	}

	// 2. ดึง token จาก "Bearer <token>"
	var accessTokenString string
	_, err := fmt.Sscanf(authHeader, "Bearer %s", &accessTokenString)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header ไม่ถูกต้อง"})
		return
	}

	// 3. Verify access token และดึง user_id
	secretKey := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(accessTokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token ไม่ถูกต้อง"})
		return
	}

	// 4. ตรวจสอบว่า token type เป็น access
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token type ไม่ถูกต้อง"})
		return
	}

	// 5. ดึง user_id
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ user_id ใน token"})
		return
	}
	userID := uint(userIDFloat)

	// 6. ยกเลิก refresh token ของผู้ใช้นี้
	// ถ้ามี device_id ส่งมา จะยกเลิกเฉพาะเครื่องนั้น ถ้าไม่ส่งมา (client รุ่นเก่า) จะยกเลิกทุกเครื่อง
	now := nowInBangkok()
	query := database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID)
	if body.DeviceID != "" {
		query = query.Where("device_id = ?", body.DeviceID)
	}
	if err := query.Updates(map[string]interface{}{
		"revoked_at":     now,
		"revoked_reason": "logout",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถออกจากระบบได้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ออกจากระบบสำเร็จ",
	})
}
