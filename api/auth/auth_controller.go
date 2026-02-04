package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt" // <--- Import ตัวนี้เพิ่มมา

	// 👇 แก้ Path ให้ตรงกับโฟลเดอร์ในเครื่องของคุณ
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔥 1. เข้ารหัส Password ก่อนบันทึก (Hash)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14) // 14 คือความยาก (Cost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถเข้ารหัสรหัสผ่านได้"})
		return
	}

	user := models.User{
		Username: input.Username,
		Password: string(hashedPassword), // 🔥 บันทึกตัวที่ Hash แล้ว
		Email:    input.Email,
		Phone:    input.Phone,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Username หรือ เบอร์โทร นี้ถูกใช้งานแล้ว"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Register Success", "data": user})
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
func Login(c *gin.Context) {
	var input models.LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรอกข้อมูลไม่ครบ"})
		return
	}

	var user models.User

	// 1. ค้นหา User (ด้วย Email หรือ Phone)
	if err := database.DB.Where("email = ? OR phone = ?", input.Username, input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้งาน"})
		return
	}

	// 🔥 2. เช็ค Password แบบ Hash (bcrypt)
	// เอา (รหัสใน DB, รหัสที่กรอกเข้ามา) มาเทียบกัน
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	
	if err != nil {
		// ถ้า err ไม่เป็น nil แปลว่ารหัสผิด
		c.JSON(http.StatusUnauthorized, gin.H{"error": "รหัสผ่านไม่ถูกต้อง"})
		return
	}

	// 3. สร้าง Token (เหมือนเดิม)
	claims := jwt.MapClaims{
		"user_id":  user.UserID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey := os.Getenv("JWT_SECRET")

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "สร้าง Token ไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login Success",
		"token":   tokenString,
		"user": gin.H{
			"id":       user.UserID,
			"username": user.Username,
			"email":    user.Email,
			"phone":    user.Phone,
		},
	})
}