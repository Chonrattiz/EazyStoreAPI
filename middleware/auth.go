package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ฟังก์ชันสำหรับตรวจเช็ค Token
func CheckAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. ดึง Header ที่ชื่อ Authorization
        authHeader := c.GetHeader("Authorization")

        // 2. เช็คว่าส่งมาไหม และรูปแบบถูกต้องไหม (ต้องขึ้นต้นด้วย Bearer )
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "กรุณาเข้าสู่ระบบก่อนใช้งาน (No Token)"})
            c.Abort() // หยุดการทำงานทันที ไม่ให้ไปต่อ
            return
        }

        // 3. ตัดคำว่า "Bearer " ออก ให้เหลือแต่ตัว Token เพียวๆ
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // 4. ตรวจสอบความถูกต้องของ Token (Verify)
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // เช็คว่าวิธีเข้ารหัสตรงกันไหม (HS256)
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            // *สำคัญ* ต้องใช้ Secret Key ตัวเดียวกับตอน Login เป๊ะๆ
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Session หมดอายุ หรือ Token ไม่ถูกต้อง"})
            c.Abort()
            return
        }

        // 5. ถ้าผ่าน! ดึงข้อมูล User จาก Token มาแปะไว้ใน Context
        // เพื่อให้ Controller เอาไปใช้ต่อได้ (เช่น รู้ว่าใครเป็นคนยิงมา)
        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            c.Set("user_id", claims["user_id"])
            c.Set("username", claims["username"])
        }

        c.Next() // อนุญาตให้ไปทำฟังก์ชันถัดไปได้ (เช่น ไป CreateShop)
    }
}