package routes

import (
	resetController "EazyStoreAPI/api/ResetPassword"
	authController "EazyStoreAPI/api/auth"
	productController "EazyStoreAPI/api/products"
	shopController "EazyStoreAPI/api/shops"

	"EazyStoreAPI/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// -----------------------------------------------------
	// 🟢 โซน Public (ไม่ต้องใช้ Token)
	// -----------------------------------------------------
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)
        auth.POST("/verify-registration", authController.VerifyRegistration)
        auth.POST("/change-email-verify", authController.ChangeEmailBeforeVerify)

        
		//  เพิ่มเส้นทางสำหรับกู้รหัสผ่านตรงนี้ครับ
		auth.POST("/request-reset", resetController.RequestResetOTP)
		auth.POST("/verify-otp", resetController.VerifyOTP)
		auth.POST("/reset-password", resetController.UpdatePassword)
	}

	// -----------------------------------------------------
	//  โซน Protected (ต้องมี Token เท่านั้นถึงจะเข้าได้)
	// -----------------------------------------------------
	// สร้างกลุ่ม api ใหม่ แล้วสั่ง Use(middleware.CheckAuth())
	protected := r.Group("/api")
	protected.Use(middleware.CheckAuth())
	{

		protected.POST("/createShop", shopController.CreateShop)

		protected.POST("/createProduct", productController.CreateProduct)
		protected.GET("/categories", productController.GetCategories)

		// ทดสอบระบบ (Test Token)
		protected.GET("/profile", func(c *gin.Context) {
			// ลองดึงค่าที่ Middleware แปะไว้ให้ออกมาดู
			userId, _ := c.Get("user_id")
			username, _ := c.Get("username")

			c.JSON(200, gin.H{
				"message":   "คุณเข้าสู่โซนปลอดภัยได้แล้ว!",
				"your_id":   userId,
				"your_name": username,
			})
		})
	}

	return r
}
