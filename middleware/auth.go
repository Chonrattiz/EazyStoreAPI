package middleware

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "กรุณาเข้าสู่ระบบก่อนใช้งาน (No Token)"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		//ตรวจสอบความถูกต้องของ Token (Verify)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session หมดอายุ หรือ Token ไม่ถูกต้อง"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
		}

		c.Next() // อนุญาตให้ไปทำฟังก์ชันถัดไปได้ (เช่น ไป CreateShop)
	}
}

var ErrNoShop = errors.New("ไม่พบข้อมูลร้านค้าของผู้ใช้")
var ErrShopForbidden = errors.New("ไม่มีสิทธิ์เข้าถึงข้อมูลของร้านค้านี้")

const shopIDsContextKey = "auth_shop_ids"

// GetShopIDsFromAuth คืน shop_id ทุกร้านที่ user ที่ล็อกอินอยู่เป็นเจ้าของ
//
// ใช้ shops.user_id เป็นแหล่งข้อมูลหลัก (CreateShop เป็นคนเขียนค่านี้) และรวม
// users.shop_id เข้าไปด้วยเผื่อบัญชีเก่าที่ผูกไว้ทางคอลัมน์นั้นอย่างเดียว
// ทั้งสองทางเป็นข้อมูลฝั่งเซิร์ฟเวอร์ที่อิง user ที่ยืนยันตัวตนแล้ว จึงเชื่อถือได้
func GetShopIDsFromAuth(c *gin.Context) ([]int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, ErrNoShop
	}

	// cache ไว้ใน context เพราะ handler เดียวอาจเรียกซ้ำหลายรอบ จะได้ไม่ query ซ้ำ
	if cached, ok := c.Get(shopIDsContextKey); ok {
		if ids, ok := cached.([]int); ok {
			return ids, nil
		}
	}

	var shopIDs []int
	if err := database.DB.Model(&models.Shop{}).
		Where("user_id = ?", userID).
		Pluck("shop_id", &shopIDs).Error; err != nil {
		return nil, err
	}

	var user models.User
	if err := database.DB.Where("user_id = ?", userID).First(&user).Error; err == nil && user.ShopID != 0 {
		found := false
		for _, id := range shopIDs {
			if id == user.ShopID {
				found = true
				break
			}
		}
		if !found {
			shopIDs = append(shopIDs, user.ShopID)
		}
	}

	if len(shopIDs) == 0 {
		return nil, ErrNoShop
	}

	c.Set(shopIDsContextKey, shopIDs)
	return shopIDs, nil
}

// GetShopIDFromAuth ดึง shopId ของ authenticated user จากฐานข้อมูล
// ใช้กับ endpoint ที่ client ไม่ได้ส่ง shop_id มา (เช่นอ้างอิงด้วย product_id อย่างเดียว)
//
// สำคัญ: ต้องคืน error เสมอเมื่อหา shop ไม่ได้ ห้ามคืน (0, nil)
// เพราะ caller ที่เช็คแค่ err จะได้ shopID = 0 แล้ว query หลุด scope ร้าน
func GetShopIDFromAuth(c *gin.Context) (int, error) {
	shopIDs, err := GetShopIDsFromAuth(c)
	if err != nil {
		return 0, err
	}
	return shopIDs[0], nil
}

// AuthorizeShopAccess ตรวจว่า shopID ที่ client ส่งมาเป็นร้านของ user ที่ล็อกอินอยู่จริงไหม
func AuthorizeShopAccess(c *gin.Context, shopID int) error {
	if shopID <= 0 {
		return ErrShopForbidden
	}

	shopIDs, err := GetShopIDsFromAuth(c)
	if err != nil {
		return err
	}

	for _, id := range shopIDs {
		if id == shopID {
			return nil
		}
	}

	return ErrShopForbidden
}

// RequireShopAccess ตรวจสิทธิ์ shopID (ที่มาจาก request body) แล้วตอบ error กลับให้เลยถ้าไม่ผ่าน
// คืน false เมื่อ response ถูกเขียนไปแล้ว — caller ต้อง return ทันที
func RequireShopAccess(c *gin.Context, shopID int) bool {
	if err := AuthorizeShopAccess(c, shopID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": ErrShopForbidden.Error()})
		return false
	}
	return true
}

// RequireShopIDQuery อ่าน shop_id จาก query string แล้วตรวจสิทธิ์ให้ในตัว
// คืน false เมื่อ response ถูกเขียนไปแล้ว — caller ต้อง return ทันที
func RequireShopIDQuery(c *gin.Context) (int, bool) {
	shopIDStr := c.Query("shop_id")
	if shopIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
		return 0, false
	}

	shopID, err := strconv.Atoi(shopIDStr)
	if err != nil || shopID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shop_id ไม่ถูกต้อง"})
		return 0, false
	}

	if !RequireShopAccess(c, shopID) {
		return 0, false
	}

	return shopID, true
}
