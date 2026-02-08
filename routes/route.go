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


	// โซน Public (ไม่ต้องใช้ Token)
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

	//  โซน Protected (ต้องมี Token เท่านั้นถึงจะเข้าได้)

	protected := r.Group("/api")
	protected.Use(middleware.CheckAuth())
	{

		protected.POST("/createShop", shopController.CreateShop)
		protected.GET("/getShop", shopController.GetShopByUser)
		protected.DELETE("/deleteShop/:shop_id", shopController.DeleteShop)
		protected.PUT("/updateShop/:shop_id", shopController.UpdateShop)

		protected.POST("/createProduct", productController.CreateProduct)
		protected.GET("/categories", productController.GetCategories)


	}

	return r
}
