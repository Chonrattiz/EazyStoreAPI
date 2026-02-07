package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt" // <--- Import ตัวนี้เพิ่มมา

	// 👇 แก้ Path ให้ตรงกับโฟลเดอร์ในเครื่องของคุณ
	resetController "EazyStoreAPI/api/ResetPassword"
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

    // 1. ตรวจสอบว่ามี Username หรือเบอร์นี้อยู่แล้วหรือไม่
    var existingUser models.User
    result := database.DB.Where("username = ? OR phone = ?", input.Username, input.Phone).First(&existingUser)

    if result.Error == nil {
        // เคสที่ 1: มี User นี้แล้ว และยืนยันตัวตนสำเร็จแล้ว -> ห้ามสมัครซ้ำ
        if existingUser.IsVerified {
            c.JSON(http.StatusConflict, gin.H{"error": "Username หรือ เบอร์โทรนี้ถูกใช้งานแล้ว"})
            return
        }
        
        // เคสที่ 2: มี User แล้วแต่ยังไม่ยืนยัน -> อนุญาตให้อัปเดตข้อมูล (เช่น เปลี่ยนอีเมล)
        hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
        existingUser.Email = input.Email
        existingUser.Password = string(hashedPassword)
        database.DB.Save(&existingUser)
    } else {
        // เคสที่ 3: ยังไม่มี User นี้เลย -> สร้างใหม่ตามปกติ
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

    // 🔥 ส่ง/อัปเดต OTP ใหม่เสมอ
    otp := resetController.GenerateOTP()
    verification := models.EmailVerification{
        Email:     input.Email,
        OTPCode:   otp,
        ExpiresAt: time.Now().Add(15 * time.Minute),
    }
    database.DB.Save(&verification)

    go resetController.SendEmailOTP(input.Email, otp)

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
    // 1. ตรวจสอบรหัสจากตาราง email_verifications
    if err := database.DB.Where("email = ? AND otp_code = ?", input.Email, input.OTP).First(&record).Error; err != nil {
        c.JSON(400, gin.H{"error": "รหัส OTP ไม่ถูกต้อง หรือหมดอายุ"})
        return
    }

    // 2. ตรวจสอบเวลาหมดอายุ
    if time.Now().After(record.ExpiresAt) {
        c.JSON(400, gin.H{"error": "รหัส OTP หมดอายุแล้ว"})
        return
    }

    // 3. ปลดล็อก User (เปลี่ยน is_verified เป็น true)
    database.DB.Model(&models.User{}).Where("email = ?", input.Email).Update("is_verified", true)
    
    // 4. ลบ OTP ทิ้ง
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

    // ค้นหา User ที่ยังไม่ยืนยัน
    var user models.User
    if err := database.DB.Where("username = ? AND is_verified = ?", input.Username, false).First(&user).Error; err != nil {
        c.JSON(404, gin.H{"error": "ไม่พบข้อมูลผู้ใช้ที่รอการยืนยัน"})
        return
    }

    // อัปเดตอีเมลใหม่
    user.Email = input.NewEmail
    database.DB.Save(&user)

    // ส่ง OTP ใหม่ไปที่เมลใหม่
    otp := resetController.GenerateOTP()
    verification := models.EmailVerification{
        Email:     input.NewEmail,
        OTPCode:   otp,
        ExpiresAt: time.Now().Add(15 * time.Minute),
    }
    database.DB.Save(&verification)

    go resetController.SendEmailOTP(input.NewEmail, otp)

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

    // 1. ตรวจสอบว่ากรอกข้อมูลมาครบถ้วนหรือไม่
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณากรอกข้อมูลให้ครบถ้วน"})
        return
    }

    var user models.User
    // ข้อความ Error แบบกลางเพื่อความปลอดภัย (User Enumeration Protection)
    invalidMsg := "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง"

    // 2. ค้นหาผู้ใช้งานด้วย Email หรือเบอร์โทรศัพท์
    if err := database.DB.Where("email = ? OR phone = ?", input.Username, input.Username).First(&user).Error; err != nil {
        // หากไม่พบผู้ใช้ ให้ตอบกลับด้วยข้อความกลาง
        c.JSON(http.StatusUnauthorized, gin.H{"error": invalidMsg})
        return
    }

    // 3. ✨ ตรวจสอบว่าผู้ใช้งานยืนยันอีเมล (OTP) แล้วหรือยัง
    if !user.IsVerified {
    c.JSON(http.StatusForbidden, gin.H{
        "error":       "กรุณายืนยันตัวตนผ่าน OTP ก่อนเข้าสู่ระบบ",
        "is_verified": false,
        "email":       user.Email,    // ส่งเมลจริงจาก DB
        "username":    user.Username, // ✨ ต้องส่ง Username จริงกลับไปด้วย!
    })
    return
}

    // 4. ตรวจสอบรหัสผ่านโดยใช้ Bcrypt
    err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
    if err != nil {
        // หากรหัสผ่านผิด ให้ตอบกลับด้วยข้อความกลาง
        c.JSON(http.StatusUnauthorized, gin.H{"error": invalidMsg})
        return
    }

    // 5. สร้าง JWT Token (มีอายุ 24 ชั่วโมง)
    claims := jwt.MapClaims{
        "user_id":  user.UserID,
        "username": user.Username,
        "exp":      time.Now().Add(time.Hour * 24).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    secretKey := os.Getenv("JWT_SECRET")

    tokenString, err := token.SignedString([]byte(secretKey))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างรหัสการเข้าถึงได้"})
        return
    }

    // 6. ส่งข้อมูลกลับเมื่อ Login สำเร็จ
    c.JSON(http.StatusOK, gin.H{
        "message": "เข้าสู่ระบบสำเร็จ",
        "token":   tokenString,
        "user": gin.H{
            "id":       user.UserID,
            "username": user.Username,
            "email":    user.Email,
            "phone":    user.Phone,
        },
    })
}